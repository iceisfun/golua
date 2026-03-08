package stdlib

import (
	"fmt"
	"strings"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/vm"
)

// maxLoadSize is the maximum size (in bytes) that the reader accumulator in
// load() will accept. This prevents Go OOM crashes when a reader function
// returns data indefinitely. Lua 5.4 uses MAX_SIZE (~(size_t)0 / 2) and
// relies on malloc returning NULL; Go has no such safety net, so we use a
// practical limit. 256 MB is far beyond any realistic Lua source file.
const maxLoadSize = 1 << 28 // 256 MB

// maxLoadLines is the maximum number of lines accepted during reader
// accumulation. Lua 5.4 uses MAX_INT; we use a lower limit that is still
// well beyond any realistic source file but small enough to be caught
// before the maxLoadSize byte limit fires on all-newline input.
const maxLoadLines = 1 << 26 // ~67 million lines

// openLoad registers load, loadfile, and dofile functions.
// These are only registered if a code provider is available and has the right capabilities.
func openLoad(v *vm.VM) {
	// load is always available (for loading from strings)
	v.SetGlobal("load", vm.NewNativeFunc(luaLoad))

	// loadfile and dofile depend on the code provider
	provider := v.CodeProvider()
	if provider != nil {
		caps := provider.Capabilities()
		if caps.AllowLoadfile {
			v.SetGlobal("loadfile", vm.NewNativeFunc(luaLoadfile))
		}
		if caps.AllowDofile {
			v.SetGlobal("dofile", vm.NewNativeFunc(luaDofile))
		}
	}
}

// load(chunk [, chunkname [, mode [, env]]])
// Loads a chunk. Returns the compiled function or nil + error message.
//
// chunk: string or function that returns strings
// chunkname: name for error messages (default "=(load)")
// mode: "b" (binary), "t" (text), or "bt" (default "bt", we only support "t")
// env: environment table for the loaded chunk
func luaLoad(v *vm.VM) int {
	chunk := v.Get(1)
	hasChunkName := !v.Get(2).IsNil()
	rawChunkName := "" // set below; stored verbatim in proto.Source
	if hasChunkName {
		rawChunkName = v.Get(2).AsString()
	}
	mode := "bt"
	if !v.Get(3).IsNil() {
		m := v.Get(3)
		if m.IsString() || m.IsNumber() {
			mode = valueToString(m)
		} else {
			callerArgError(v, 3, "load", fmt.Sprintf("string expected, got %s", m.Type()))
		}
	}
	env := v.Get(4)
	hasEnv := v.ArgCount() >= 4

	// Get the source code
	var source string
	if chunk.IsString() {
		source = chunk.AsString()
		if !hasChunkName {
			// Default: source text itself (matches Lua 5.4)
			rawChunkName = source
		}
	} else if chunk.IsFunction() || chunk.IsNativeFunc() {
		if !hasChunkName {
			rawChunkName = "=(load)"
		}
		// Call the function repeatedly to get the source
		var builder []byte
		lineCount := 0
		exitNonYieldable := v.EnterNonYieldable()
		defer exitNonYieldable()
		for {
			results, err := v.ProtectedCall(chunk, nil)
			if err != nil {
				v.Set(0, vm.Nil)
				// Lua 5.4: returns the raw error from the reader, not prefixed.
				// For non-string errors, format as "(error object is a TYPE value)".
				if le, ok := err.(*vm.LuaError); ok {
					if le.Value.IsString() {
						v.Set(1, vm.NewString(le.Value.AsString()))
					} else {
						v.Set(1, vm.NewString(fmt.Sprintf("(error object is a %s value)", le.Value.Type())))
					}
				} else {
					v.Set(1, vm.NewString(err.Error()))
				}
				return 2
			}
			if len(results) == 0 || results[0].IsNil() {
				break
			}
			var s string
			if results[0].IsString() || results[0].IsNumber() {
				s = valueToString(results[0])
			} else {
				v.Set(0, vm.Nil)
				v.Set(1, vm.NewString(v.AddCallerLocation("reader function must return a string")))
				return 2
			}
			if s == "" {
				break
			}
			lineCount += strings.Count(s, "\n")
			if lineCount >= maxLoadLines {
				v.Set(0, vm.Nil)
				chName := chunkNameForDisplay(rawChunkName)
				v.Set(1, vm.NewString(fmt.Sprintf("%s: chunk has too many lines", chName)))
				return 2
			}
			if len(builder)+len(s) > maxLoadSize {
				v.Set(0, vm.Nil)
				v.Set(1, vm.NewString("not enough memory"))
				return 2
			}
			builder = append(builder, s...)
		}
		source = string(builder)
	} else if chunk.IsNumber() {
		source = valueToString(chunk)
		if !hasChunkName {
			rawChunkName = source
		}
	} else {
		got := chunk.Type()
		if v.ArgCount() < 1 {
			got = "no value"
		}
		callerArgError(v, 1, "load", fmt.Sprintf("function expected, got %s", got))
	}

	// Detect binary chunk (starts with \x1bLua)
	isBinary := len(source) >= 4 && source[0] == '\x1b' && source[1:4] == "Lua"

	if isBinary {
		if !strings.Contains(mode, "b") {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(fmt.Sprintf("attempt to load a binary chunk (mode is '%s')", mode)))
			return 2
		}
		fn, errMsg := loadBinaryChunk(v, source, rawChunkName, env, hasEnv)
		if errMsg != "" {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(errMsg))
			return 2
		}
		v.Set(0, fn)
		return 1
	}

	// Text chunk
	if !strings.Contains(mode, "t") {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(fmt.Sprintf("attempt to load a text chunk (mode is '%s')", mode)))
		return 2
	}

	// Format chunkname for parser/compiler error messages (shortSrc-style):
	// "=xxx" → "xxx", "@xxx" → "xxx", else → [string "xxx"]
	displayName := chunkNameForDisplay(rawChunkName)

	// Parse and compile
	fn, errMsg := compileChunk(v, source, displayName, env, hasEnv, compileChunkOpts{rawSource: rawChunkName, hasRawSource: true})
	if errMsg != "" {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(errMsg))
		return 2
	}

	v.Set(0, fn)
	return 1
}

// chunkNameForDisplay formats a raw chunkname into the display form used by
// the parser/compiler for error messages. This mirrors Lua 5.4's luaO_chunkid.
func chunkNameForDisplay(name string) string {
	if len(name) == 0 {
		return `[string ""]`
	}
	switch name[0] {
	case '=':
		s := name[1:]
		if len(s) > 59 {
			s = s[:59]
		}
		return s
	case '@':
		s := name[1:]
		if len(s) >= 60 {
			s = "..." + s[len(s)-56:]
		}
		return s
	default:
		s := name
		truncated := false
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
			truncated = true
		}
		if len(s) >= 45 {
			s = s[:45]
			truncated = true
		}
		if truncated {
			return fmt.Sprintf(`[string "%s..."]`, s)
		}
		return fmt.Sprintf(`[string "%s"]`, s)
	}
}

// loadfile([filename [, mode [, env]]])
// Loads a chunk from a file via the code provider.
// Returns the compiled function or nil + error message.
func luaLoadfile(v *vm.VM) int {
	provider := v.CodeProvider()
	if provider == nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString("loadfile not available: no code provider configured"))
		return 2
	}

	caps := provider.Capabilities()
	if !caps.AllowLoadfile {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString("loadfile not permitted by code provider"))
		return 2
	}

	// Get filename
	filename := ""
	if !v.Get(1).IsNil() {
		filename = v.Get(1).AsString()
	}
	if filename == "" {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString("bad argument #1 to 'loadfile' (filename expected)"))
		return 2
	}

	mode := "bt"
	if !v.Get(2).IsNil() {
		m := v.Get(2)
		if m.IsString() || m.IsNumber() {
			mode = valueToString(m)
		} else {
			callerArgError(v, 2, "loadfile", fmt.Sprintf("string expected, got %s", m.Type()))
		}
	}
	env := v.Get(3)
	hasEnv := v.ArgCount() >= 3

	// Load the source via the code provider
	ctx := v.CallerContext()
	source, chunkName, err := provider.LoadChunk(filename, ctx)
	if err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}

	if !strings.Contains(mode, "t") {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(fmt.Sprintf("attempt to load a text chunk (mode is '%s')", mode)))
		return 2
	}

	// Parse and compile (loadfile should strip shebangs)
	// chunkName from the provider is already formatted (e.g. "@filename")
	fn, errMsg := compileChunk(v, string(source), chunkName, env, hasEnv, compileChunkOpts{stripShebang: true, rawSource: chunkName})
	if errMsg != "" {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(errMsg))
		return 2
	}

	v.Set(0, fn)
	return 1
}

// dofile([filename])
// Opens and executes the file immediately.
// Returns the values returned by the chunk.
func luaDofile(v *vm.VM) int {
	provider := v.CodeProvider()
	if provider == nil {
		panic("dofile not available: no code provider configured")
	}

	caps := provider.Capabilities()
	if !caps.AllowDofile {
		panic("dofile not permitted by code provider")
	}

	// Get filename
	filename := ""
	if !v.Get(1).IsNil() {
		filename = v.Get(1).AsString()
	}
	if filename == "" {
		panic("bad argument #1 to 'dofile' (filename expected)")
	}

	// Load the source via the code provider
	ctx := v.CallerContext()
	source, chunkName, err := provider.LoadChunk(filename, ctx)
	if err != nil {
		panic(err.Error())
	}

	// Reuse compileChunk with shebang stripping (like loadfile)
	fn, errMsg := compileChunk(v, string(source), chunkName, vm.Nil, false, compileChunkOpts{stripShebang: true, rawSource: chunkName})
	if errMsg != "" {
		panic(errMsg)
	}

	// Save and set chunk name
	oldChunkName := v.ChunkName()
	v.SetChunkName(chunkName)
	defer v.SetChunkName(oldChunkName)

	// Execute
	results, runErr := v.ProtectedCall(fn, nil)
	if runErr != nil {
		// Preserve LuaError so pcall(dofile, ...) returns the original value
		if luaErr, ok := runErr.(*vm.LuaError); ok {
			panic(luaErr)
		}
		panic(runErr.Error())
	}

	// Return all results
	for i, r := range results {
		v.Set(i, r)
	}
	return len(results)
}

// compileChunkOpts holds optional settings for compileChunk.
type compileChunkOpts struct {
	stripShebang  bool
	rawSource     string // override proto.Source for debug info
	hasRawSource  bool   // true when rawSource is explicitly set (even if empty)
}

// compileChunk parses and compiles Lua source, returning a function value.
// If hasEnv is true, env is bound to the chunk's _ENV upvalue even when it is nil
// or non-table; otherwise globals are used.
func compileChunk(v *vm.VM, source, chunkName string, env vm.Value, hasEnv bool, opts ...compileChunkOpts) (vm.Value, string) {
	var o compileChunkOpts
	if len(opts) > 0 {
		o = opts[0]
	}

	// Strip UTF-8 BOM if present (loadfile and dofile)
	if o.stripShebang && len(source) >= 3 && source[0] == 0xEF && source[1] == 0xBB && source[2] == 0xBF {
		source = source[3:]
	}

	// Parse — use display name for parser error messages
	block, parseErr := parser.Parse(chunkName, source, o.stripShebang)
	if parseErr != nil {
		return vm.Nil, parseErr.Error()
	}

	// Compile — use raw source name so compiler.shortSrc formats correctly.
	// The display name has [string "..."] wrapping which shortSrc would double-wrap.
	compileSource := chunkName
	if o.hasRawSource {
		compileSource = o.rawSource
	}
	proto, compileErr := compiler.Compile(compileSource, block,
		compiler.WithLimits(v.GetLimits().CompilerLimits))
	if compileErr != nil {
		return vm.Nil, compileErr.Error()
	}

	// Override proto.Source with the raw source name for debug info.
	if o.hasRawSource {
		setProtoSource(proto, o.rawSource)
	}

	// Create closure
	closure := vm.NewClosure(proto)

	// Set up upvalues: first is _ENV, rest are initialized as closed nil upvalues
	for i := range proto.Upvalues {
		closure.Upvalues[i] = &vm.Upvalue{}
		if i == 0 {
			if hasEnv {
				// Use provided environment value exactly as passed.
				closure.Upvalues[0].SetClosed(env)
			} else {
				// Use global environment
				closure.Upvalues[0].SetClosed(vm.NewTable(v.Globals()))
			}
		} else {
			closure.Upvalues[i].SetClosed(vm.Nil)
		}
	}

	return vm.NewFunction(closure), ""
}

// loadBinaryChunk loads a precompiled binary chunk via the undumper.
func loadBinaryChunk(v *vm.VM, data string, chunkName string, env vm.Value, hasEnv bool) (fn vm.Value, errMsg string) {
	// Determine display name for error messages
	name := chunkName
	if len(name) > 0 {
		if name[0] == '@' || name[0] == '=' {
			name = name[1:]
		} else if name[0] == '\x1b' {
			name = "binary string"
		}
	}

	// Recover from panics in the undumper (truncated chunks, etc.)
	defer func() {
		if r := recover(); r != nil {
			fn = vm.Nil
			errMsg = fmt.Sprintf("%v", r)
		}
	}()

	proto, _, err := compiler.Undump([]byte(data), name)
	if err != nil {
		return vm.Nil, err.Error()
	}

	// Create closure
	closure := vm.NewClosure(proto)

	// Set up upvalues: first is _ENV, rest are initialized as closed nil upvalues
	for i := range proto.Upvalues {
		closure.Upvalues[i] = &vm.Upvalue{}
		if i == 0 {
			if hasEnv {
				closure.Upvalues[0].SetClosed(env)
			} else {
				closure.Upvalues[0].SetClosed(vm.NewTable(v.Globals()))
			}
		} else {
			closure.Upvalues[i].SetClosed(vm.Nil)
		}
	}

	return vm.NewFunction(closure), ""
}

// setProtoSource recursively sets the Source field on a proto and all nested protos.
func setProtoSource(proto *compiler.Proto, source string) {
	proto.Source = source
	for _, child := range proto.Protos {
		setProtoSource(child, source)
	}
}
