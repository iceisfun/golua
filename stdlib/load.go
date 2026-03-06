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
			panic(fmt.Sprintf("bad argument #3 to 'load' (string expected, got %s)", m.Type()))
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
				v.Set(1, vm.NewString(fmt.Sprintf("error calling chunk reader: %v", err)))
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
				v.Set(1, vm.NewString("reader function must return a string"))
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
		panic(fmt.Sprintf("bad argument #1 to 'load' (function expected, got %s)", chunk.Type()))
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
	fn, errMsg := compileChunk(v, source, displayName, env, hasEnv, compileChunkOpts{rawSource: rawChunkName})
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
		return name
	}
	switch name[0] {
	case '=':
		return name[1:]
	case '@':
		return name[1:]
	default:
		s := name
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
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
			panic(fmt.Sprintf("bad argument #2 to 'loadfile' (string expected, got %s)", m.Type()))
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
	stripShebang bool
	rawSource    string // if non-empty, override proto.Source for debug info
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

	// Parse
	block, parseErr := parser.Parse(chunkName, source, o.stripShebang)
	if parseErr != nil {
		return vm.Nil, fmt.Sprintf("syntax error: %v", parseErr)
	}

	// Compile
	proto, compileErr := compiler.Compile(chunkName, block,
		compiler.WithLimits(v.GetLimits().CompilerLimits))
	if compileErr != nil {
		return vm.Nil, fmt.Sprintf("compile error: %v", compileErr)
	}

	// Override proto.Source with the raw source name for debug info.
	// The chunkName used above is formatted for error messages (shortSrc style),
	// but debug.getinfo().source should return the original chunkname.
	if o.rawSource != "" {
		setProtoSource(proto, o.rawSource)
	}

	// Create closure
	closure := vm.NewClosure(proto)

	// Set up _ENV upvalue
	if len(proto.Upvalues) > 0 {
		closure.Upvalues[0] = &vm.Upvalue{}
		if hasEnv {
			// Use provided environment value exactly as passed.
			closure.Upvalues[0].SetClosed(env)
		} else {
			// Use global environment
			closure.Upvalues[0].SetClosed(vm.NewTable(v.Globals()))
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

	// Set up _ENV upvalue
	if len(proto.Upvalues) > 0 {
		closure.Upvalues[0] = &vm.Upvalue{}
		if hasEnv {
			closure.Upvalues[0].SetClosed(env)
		} else {
			closure.Upvalues[0].SetClosed(vm.NewTable(v.Globals()))
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
