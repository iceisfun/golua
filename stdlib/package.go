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
		"debug", "utf8", "bit32", "package",
	} {
		val := v.GetGlobal(name)
		if !val.IsNil() {
			loaded.SetString(name, val)
		}
	}
	pkg.SetString("loaded", vm.NewTable(loaded))

	// package.preload — empty
	pkg.SetString("preload", vm.NewTable(vm.NewEmptyTable()))

	// package.path / package.cpath
	pkg.SetString("path", vm.NewString("?.lua;?/init.lua"))
	pkg.SetString("cpath", vm.NewString("?.so;?/init"))

	// package.config (POSIX defaults)
	pkg.SetString("config", vm.NewString("/\n;\n?\n!\n-"))

	// package.searchpath
	pkg.SetString("searchpath", vm.NewNativeFunc(luaSearchPath))

	// package.searchers
	searchers := vm.NewEmptyTable()
	searchers.SetInt(1, vm.NewNativeFunc(makePreloadSearcher(pkg)))
	searchers.SetInt(2, vm.NewNativeFunc(makeLuaFileSearcher(v, pkg)))
	searchers.SetInt(3, vm.NewNativeFunc(makeCFileSearcher(pkg)))
	pkg.SetString("searchers", vm.NewTable(searchers))

	// Register package table (before loaded snapshot, but we already set it above)
	pkgVal := vm.NewTable(pkg)
	v.SetGlobal("package", pkgVal)
	loaded.SetString("package", pkgVal)

	// require — captures pkg table via closure
	v.SetGlobal("require", vm.NewNativeFunc(makeRequire(v, pkg)))
}

// makeRequire creates the require() function with a captured reference to the
// package table. This survives reassignment of the package global (Lua 5.4 behavior).
func makeRequire(v *vm.VM, pkg *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		if v.ArgCount() < 1 {
			panic("bad argument #1 to 'require' (string expected, got no value)")
		}
		nameVal := v.Get(1)
		if !nameVal.IsString() {
			panic(fmt.Sprintf("bad argument #1 to 'require' (string expected, got %s)", nameVal.Type()))
		}
		name := nameVal.AsString()

		// Check package.loaded
		loadedVal := pkg.GetString("loaded")
		nameKey := vm.NewString(name)
		if !loadedVal.IsNil() {
			loaded := loadedVal.AsTable()
			cached := loaded.Get(nameKey)
			if !cached.IsNil() {
				v.Set(0, cached)
				return 1
			}
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
				errMsgs.WriteString("\n\t")
				errMsgs.WriteString(err.Error())
				continue
			}

			if len(results) == 0 {
				continue
			}

			// If result is a string, it's an error message
			if results[0].IsString() {
				errMsgs.WriteString(results[0].AsString())
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
					if luaErr, ok := loadErr.(*vm.LuaError); ok {
						panic(luaErr)
					}
					panic(fmt.Sprintf("error loading module '%s':\n\t%s", name, loadErr.Error()))
				}

				// Get loaded table again (might have been modified by loader)
				loadedVal = pkg.GetString("loaded")
				loaded := loadedVal.AsTable()

				// If loader returns non-nil, set package.loaded[name]
				if len(loadResults) > 0 && !loadResults[0].IsNil() {
					_ = loaded.Set(nameKey, loadResults[0])
				}

				// If package.loaded[name] is still nil, set to true
				if loaded.Get(nameKey).IsNil() {
					_ = loaded.Set(nameKey, vm.True)
				}

				v.Set(0, loaded.Get(nameKey))
				if !extra.IsNil() {
					v.Set(1, extra)
					return 2
				}
				return 1
			}
		}

		panic(&vm.LuaError{Value: vm.NewString(fmt.Sprintf("module '%s' not found:%s", name, errMsgs.String()))})
	}
}

// makePreloadSearcher returns a searcher that checks package.preload.
func makePreloadSearcher(pkg *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1).AsString()
		preloadVal := pkg.GetString("preload")
		if preloadVal.IsNil() {
			v.Set(0, vm.NewString("\n\tno field package.preload['"+name+"']"))
			return 1
		}
		preload := preloadVal.AsTable()
		loader := preload.Get(vm.NewString(name))
		if loader.IsNil() {
			v.Set(0, vm.NewString("\n\tno field package.preload['"+name+"']"))
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
		name := v.Get(1).AsString()

		pathVal := pkg.GetString("path")
		if !pathVal.IsString() {
			panic("'package.path' must be a string")
		}

		// Replace dots with directory separator
		fname := strings.ReplaceAll(name, ".", "/")
		paths := expandTemplates(fname, pathVal.AsString())

		provider := machine.CodeProvider()
		var errBuf strings.Builder

		for _, path := range paths {
			if provider == nil {
				errBuf.WriteString("\n\tno file '")
				errBuf.WriteString(path)
				errBuf.WriteString("'")
				continue
			}

			ctx := v.CallerContext()
			source, chunkName, err := provider.LoadChunk(path, ctx)
			if err != nil {
				errBuf.WriteString("\n\tno file '")
				errBuf.WriteString(path)
				errBuf.WriteString("'")
				continue
			}

			// Compile and return loader
			fn, errMsg := compileChunk(v, string(source), chunkName, vm.Nil, false, compileChunkOpts{stripShebang: true, rawSource: chunkName})
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
		name := v.Get(1).AsString()

		cpathVal := pkg.GetString("cpath")
		if !cpathVal.IsString() {
			v.Set(0, vm.Nil)
			return 1
		}

		fname := strings.ReplaceAll(name, ".", "/")
		paths := expandTemplates(fname, cpathVal.AsString())

		var errBuf strings.Builder
		for _, path := range paths {
			errBuf.WriteString("\n\tno file '")
			errBuf.WriteString(path)
			errBuf.WriteString("'")
		}

		v.Set(0, vm.NewString(errBuf.String()))
		return 1
	}
}

// expandTemplates splits a path template string by ";" and replaces "?" with name.
func expandTemplates(name, path string) []string {
	templates := strings.Split(path, ";")
	var result []string
	for _, tmpl := range templates {
		if tmpl == "" {
			continue
		}
		result = append(result, strings.ReplaceAll(tmpl, "?", name))
	}
	return result
}

// luaSearchPath implements package.searchpath(name, path [, sep [, rep]]).
func luaSearchPath(v *vm.VM) int {
	if v.ArgCount() < 2 {
		panic("bad argument #1 to 'package.searchpath' (string expected, got no value)")
	}
	name := v.Get(1).AsString()
	path := v.Get(2).AsString()

	sep := "."
	if v.ArgCount() >= 3 && !v.Get(3).IsNil() {
		sep = v.Get(3).AsString()
	}
	rep := "/"
	if v.ArgCount() >= 4 && !v.Get(4).IsNil() {
		rep = v.Get(4).AsString()
	}

	// Replace sep with rep in name
	fname := name
	if sep != "" {
		fname = strings.ReplaceAll(name, sep, rep)
	}

	paths := expandTemplates(fname, path)

	var errBuf strings.Builder
	for _, p := range paths {
		errBuf.WriteString("\n\tno file '")
		errBuf.WriteString(p)
		errBuf.WriteString("'")
	}

	v.Set(0, vm.Nil)
	v.Set(1, vm.NewString(errBuf.String()))
	return 2
}
