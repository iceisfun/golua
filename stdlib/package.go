package stdlib

import (
	"fmt"
	"strings"

	"github.com/iceisfun/golua/vm"
)

// openPackage registers the package module and require global.
// Must be called AFTER all other modules so package.loaded can snapshot them.
func openPackage(v *vm.VM) {
	pkg := vm.NewEmptyTable()

	// package.loaded — pre-populated with registered stdlib modules
	loaded := vm.NewEmptyTable()
	for _, name := range []string{
		"string", "math", "table", "io", "os", "coroutine",
		"debug", "utf8", "bit32", "package", "_G",
	} {
		val := v.GetGlobal(name)
		if !val.IsNil() {
			loaded.SetString(name, val)
		}
	}
	pkg.SetString("loaded", vm.NewTable(loaded))

	// package.preload — empty
	preload := vm.NewEmptyTable()
	pkg.SetString("preload", vm.NewTable(preload))
	registry := v.GetRegistry()
	registry.Set(vm.NewString("_LOADED"), vm.NewTable(loaded))
	registry.Set(vm.NewString("_PRELOAD"), vm.NewTable(preload))

	// package.path / package.cpath
	pkg.SetString("path", vm.NewString("?.lua;?/init.lua"))
	pkg.SetString("cpath", vm.NewString("?.so;?/init"))

	// package.config (POSIX defaults)
	pkg.SetString("config", vm.NewString("/\n;\n?\n!\n-\n"))

	// package.searchpath
	pkg.SetString("searchpath", vm.NewNativeFunc(luaSearchPath))

	// package.loadlib
	pkg.SetString("loadlib", vm.NewNativeFunc(makePackageLoadlib(v.LoadLibProvider())))

	// package.searchers
	searchers := vm.NewEmptyTable()
	searchers.SetInt(1, vm.NewNativeFunc(makePreloadSearcher(preload)))
	searchers.SetInt(2, vm.NewNativeFunc(makeLuaFileSearcher(v, pkg)))
	searchers.SetInt(3, vm.NewNativeFunc(makeCFileSearcher(pkg)))
	searchers.SetInt(4, vm.NewNativeFunc(makeCRootSearcher(pkg)))
	pkg.SetString("searchers", vm.NewTable(searchers))

	// Register package table (before loaded snapshot, but we already set it above)
	pkgVal := vm.NewTable(pkg)
	v.SetGlobal("package", pkgVal)
	loaded.SetString("package", pkgVal)

	// require — captures pkg table via closure
	v.SetGlobal("require", vm.NewNativeFunc(makeRequire(v, pkg, loaded)))
}

// makePackageLoadlib creates package.loadlib(path, init).
func makePackageLoadlib(provider vm.LuaLoadLibProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		if v.ArgCount() < 1 {
			callerArgError(v, 1, "package.loadlib", "string expected, got no value")
		}
		if v.ArgCount() < 2 {
			callerArgError(v, 2, "package.loadlib", "string expected, got no value")
		}

		path := getString(v, 1, "package.loadlib")
		init := getString(v, 2, "package.loadlib")

		if provider == nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString("dynamic libraries not enabled; check your Lua installation"))
			v.Set(2, vm.NewString("absent"))
			return 3
		}

		loader, errmsg, where := provider.LoadLib(v.Context(), path, init, v.CallerContext())
		if loader == nil {
			if errmsg == "" {
				errmsg = "dynamic library loader returned no loader"
			}
			if where == "" {
				where = "open"
			}
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(errmsg))
			v.Set(2, vm.NewString(where))
			return 3
		}

		v.Set(0, vm.NewNativeFunc(loader))
		return 1
	}
}

// makeRequire creates the require() function with a captured reference to the
// package table. This survives reassignment of the package global (Lua 5.4 behavior).
func makeRequire(v *vm.VM, pkg *vm.Table, loaded *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		if v.ArgCount() < 1 {
			callerArgError(v, 1, "require", "string expected, got no value")
		}
		nameVal := v.Get(1)
		if !nameVal.IsString() && !nameVal.IsNumber() {
			callerArgError(v, 1, "require", fmt.Sprintf("string expected, got %s", nameVal.Type()))
		}
		var name string
		if nameVal.IsString() {
			name = nameVal.AsString()
		} else {
			name = vm.ValueToString(nameVal)
		}

		// Check package.loaded
		nameKey := vm.NewString(name)
		cached := loaded.Get(nameKey)
		if cached.ToBool() {
			v.Set(0, cached)
			return 1
		}

		// Validate package.searchers
		searchersVal := pkg.GetString("searchers")
		if searchersVal.IsNil() || searchersVal.Type() != "table" {
			panic("'package.searchers' must be a table")
		}
		searchersTbl := searchersVal.AsTable()

		exitNonYieldable := v.EnterNonYieldable()
		defer exitNonYieldable()

		// Iterate searchers
		var errMsgs strings.Builder
		for i := 1; i <= searchersTbl.Len(); i++ {
			searcher := searchersTbl.Get(vm.NewInt(int64(i)))
			if searcher.IsNil() {
				break
			}

			results, err := v.ProtectedCall(searcher, []vm.Value{vm.NewString(name)})
			if err != nil {
				if luaErr, ok := err.(*vm.LuaError); ok {
					panic(luaErr)
				}
				panic(err.Error())
			}

			if len(results) == 0 {
				continue
			}

			// If result is a string/number, it's an error message.
			// Match Lua 5.4 behavior: numeric results are stringified and appended,
			// while other non-function values are ignored.
			if results[0].IsString() || results[0].IsNumber() {
				errMsgs.WriteString("\n\t")
				errMsgs.WriteString(vm.ValueToString(results[0]))
				continue
			}

			// If result is a function, it's a loader
			if results[0].IsFunction() || results[0].IsNativeFunc() {
				loader := results[0]
				var extra vm.Value
				if len(results) > 1 {
					extra = results[1]
				} else {
					extra = vm.Nil
				}

				// Call loader(name, extra)
				args := []vm.Value{vm.NewString(name), extra}
				loadResults, loadErr := v.ProtectedCall(loader, args)
				if loadErr != nil {
					// Reference Lua uses lua_call (unprotected) here; runtime
					// errors from the loaded chunk propagate unchanged. The
					// "error loading module ... from file ..." prefix is added
					// by the file searcher (checkload in C) only when the
					// load/parse phase fails. Re-raise the error as-is.
					if luaErr, ok := loadErr.(*vm.LuaError); ok {
						panic(luaErr)
					}
					panic(&vm.LuaError{Value: vm.NewString(loadErr.Error())})
				}

				// If loader returns non-nil, set package.loaded[name]
				if len(loadResults) > 0 && !loadResults[0].IsNil() {
					_ = loaded.Set(nameKey, loadResults[0])
				}

				// If package.loaded[name] is still nil, set to true
				if loaded.Get(nameKey).IsNil() {
					_ = loaded.Set(nameKey, vm.True)
				}

				v.Set(0, loaded.Get(nameKey))
				v.Set(1, extra)
				return 2
			}
		}

		panic(&vm.LuaError{Value: vm.NewString(fmt.Sprintf("module '%s' not found:%s", name, errMsgs.String()))})
	}
}

// makePreloadSearcher returns a searcher that checks package.preload.
func makePreloadSearcher(preload *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := getString(v, 1, "?")
		loader := preload.Get(vm.NewString(name))
		if loader.IsNil() {
			v.Set(0, vm.NewString("no field package.preload['"+name+"']"))
			return 1
		}
		v.Set(0, loader)
		v.Set(1, vm.NewString(":preload:"))
		return 2
	}
}

// makeLuaFileSearcher returns a searcher that tries to load Lua files via the code provider.
func makeLuaFileSearcher(machine *vm.VM, pkg *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := getString(v, 1, "?")

		pathVal := pkg.GetString("path")
		pathStr, ok := coercePathString(pathVal)
		if !ok {
			panic("'package.path' must be a string")
		}

		// Replace dots with directory separator
		fname := strings.ReplaceAll(name, ".", "/")
		paths := expandTemplates(fname, pathStr)

		provider := machine.CodeProvider()
		var errBuf strings.Builder

		for _, path := range paths {
			if provider == nil {
				appendNoFileErr(&errBuf, path)
				continue
			}

			ctx := v.CallerContext()
			source, chunkName, err := provider.LoadChunk(v.Context(), path, ctx)
			if err != nil {
				appendNoFileErr(&errBuf, path)
				continue
			}

			// Compile and return loader (detect binary chunks)
			var fn vm.Value
			var errMsg string
			sourceStr := string(source)
			if len(sourceStr) > 0 && sourceStr[0] == '\x1b' {
				fn, errMsg = loadBinaryChunk(v, sourceStr, chunkName, vm.Nil, false)
			} else {
				displayName := chunkNameForDisplay(chunkName)
				fn, errMsg = compileChunk(v, sourceStr, displayName, vm.Nil, false, compileChunkOpts{stripShebang: true, rawSource: chunkName, hasRawSource: true})
			}
			if errMsg != "" {
				panic(fmt.Sprintf("error loading module '%s' from file '%s':\n\t%s", name, path, errMsg))
			}

			v.Set(0, fn)
			v.Set(1, vm.NewString(path))
			return 2
		}

		v.Set(0, vm.NewString(errBuf.String()))
		return 1
	}
}

// makeCFileSearcher returns a searcher that always fails (C loading not supported).
func makeCFileSearcher(pkg *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := getString(v, 1, "?")

		cpathVal := pkg.GetString("cpath")
		cpathStr, ok := coercePathString(cpathVal)
		if !ok {
			v.Set(0, vm.Nil)
			return 1
		}

		fname := strings.ReplaceAll(name, ".", "/")
		paths := expandTemplates(fname, cpathStr)

		var errBuf strings.Builder
		for _, path := range paths {
			appendNoFileErr(&errBuf, path)
		}

		v.Set(0, vm.NewString(errBuf.String()))
		return 1
	}
}

// makeCRootSearcher returns a searcher for the root module of dotted names.
// GoLua does not load C modules here, but Lua 5.4 still exposes the searcher
// and includes its path probes in module-not-found errors.
func makeCRootSearcher(pkg *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := getString(v, 1, "?")
		dot := strings.IndexByte(name, '.')
		if dot < 0 {
			return 0
		}

		cpathVal := pkg.GetString("cpath")
		cpathStr, ok := coercePathString(cpathVal)
		if !ok {
			v.Set(0, vm.Nil)
			return 1
		}

		root := name[:dot]
		paths := expandTemplates(strings.ReplaceAll(root, ".", "/"), cpathStr)

		var errBuf strings.Builder
		for _, path := range paths {
			appendNoFileErr(&errBuf, path)
		}

		v.Set(0, vm.NewString(errBuf.String()))
		return 1
	}
}

// coercePathString accepts package.path / package.cpath values, coercing
// numbers to their string form (matching reference Lua's lua_tostring on the
// path field). Returns the coerced string and ok=true on success; ok=false for
// any other type (including nil) so callers can raise the appropriate error.
func coercePathString(val vm.Value) (string, bool) {
	if val.IsString() {
		return val.AsString(), true
	}
	if val.IsNumber() {
		return vm.ValueToString(val), true
	}
	return "", false
}

// expandTemplates splits a path template string by ";" and replaces "?" with name.
func expandTemplates(name, path string) []string {
	templates := strings.Split(path, ";")
	result := make([]string, 0, len(templates))
	for _, tmpl := range templates {
		result = append(result, strings.ReplaceAll(tmpl, "?", name))
	}
	return result
}

func appendNoFileErr(errBuf *strings.Builder, path string) {
	if errBuf.Len() > 0 {
		errBuf.WriteString("\n\t")
	}
	errBuf.WriteString("no file '")
	errBuf.WriteString(path)
	errBuf.WriteString("'")
}

// luaSearchPath implements package.searchpath(name, path [, sep [, rep]]).
func luaSearchPath(v *vm.VM) int {
	// Match Lua 5.4 argument error order by validating #2 (path) before #1 (name).
	path := getString(v, 2, "package.searchpath")
	name := getString(v, 1, "package.searchpath")

	sep := "."
	if v.ArgCount() >= 3 && !v.Get(3).IsNil() {
		sep = getString(v, 3, "package.searchpath")
	}
	rep := "/"
	if v.ArgCount() >= 4 && !v.Get(4).IsNil() {
		rep = getString(v, 4, "package.searchpath")
	}

	// Replace sep with rep in name
	fname := name
	if sep != "" {
		fname = strings.ReplaceAll(name, sep, rep)
	}

	paths := expandTemplates(fname, path)
	provider := v.CodeProvider()
	var caller *vm.LuaCallerContext
	if provider != nil {
		caller = v.CallerContext()
	}

	var errBuf strings.Builder
	for _, p := range paths {
		if provider != nil {
			if _, _, err := provider.LoadChunk(v.Context(), p, caller); err == nil {
				v.Set(0, vm.NewString(p))
				return 1
			}
		}
		appendNoFileErr(&errBuf, p)
	}

	v.Set(0, vm.Nil)
	v.Set(1, vm.NewString(errBuf.String()))
	return 2
}
