package stdlib

import (
	"fmt"
	"strings"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/vm"
)

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
	chunkName := "=(load)"
	if !v.Get(2).IsNil() {
		chunkName = v.Get(2).AsString()
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

	// Get the source code
	var source string
	if chunk.IsString() {
		source = chunk.AsString()
	} else if chunk.IsFunction() || chunk.IsNativeFunc() {
		// Call the function repeatedly to get the source
		var builder []byte
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
			builder = append(builder, s...)
		}
		source = string(builder)
	} else {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString("bad argument #1 to 'load' (string or function expected)"))
		return 2
	}

	if !strings.Contains(mode, "t") {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(fmt.Sprintf("attempt to load a text chunk (mode is '%s')", mode)))
		return 2
	}

	// Parse and compile
	fn, errMsg := compileChunk(v, source, chunkName, env)
	if errMsg != "" {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(errMsg))
		return 2
	}

	v.Set(0, fn)
	return 1
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

	// Parse and compile
	fn, errMsg := compileChunk(v, string(source), chunkName, env)
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

	// Parse
	block, parseErr := parser.Parse(chunkName, string(source))
	if parseErr != nil {
		panic(fmt.Sprintf("syntax error: %v", parseErr))
	}

	// Compile
	proto, compileErr := compiler.Compile(chunkName, block)
	if compileErr != nil {
		panic(fmt.Sprintf("compile error: %v", compileErr))
	}

	// Create closure with _ENV pointing to globals
	closure := vm.NewClosure(proto)
	if len(proto.Upvalues) > 0 {
		closure.Upvalues[0] = &vm.Upvalue{}
		closure.Upvalues[0].SetClosed(vm.NewTable(v.Globals()))
	}

	// Save and set chunk name
	oldChunkName := v.ChunkName()
	v.SetChunkName(chunkName)
	defer v.SetChunkName(oldChunkName)

	// Execute
	results, runErr := v.ProtectedCall(vm.NewFunction(closure), nil)
	if runErr != nil {
		panic(runErr.Error())
	}

	// Return all results
	for i, r := range results {
		v.Set(i, r)
	}
	return len(results)
}

// compileChunk parses and compiles Lua source, returning a function value.
// If env is non-nil, it's used as the environment for the chunk.
func compileChunk(v *vm.VM, source, chunkName string, env vm.Value) (vm.Value, string) {
	// Parse
	block, parseErr := parser.Parse(chunkName, source)
	if parseErr != nil {
		return vm.Nil, fmt.Sprintf("syntax error: %v", parseErr)
	}

	// Compile
	proto, compileErr := compiler.Compile(chunkName, block)
	if compileErr != nil {
		return vm.Nil, fmt.Sprintf("compile error: %v", compileErr)
	}

	// Create closure
	closure := vm.NewClosure(proto)

	// Set up _ENV upvalue
	if len(proto.Upvalues) > 0 {
		closure.Upvalues[0] = &vm.Upvalue{}
		if !env.IsNil() && env.IsTable() {
			// Use provided environment
			closure.Upvalues[0].SetClosed(env)
		} else {
			// Use global environment
			closure.Upvalues[0].SetClosed(vm.NewTable(v.Globals()))
		}
	}

	return vm.NewFunction(closure), ""
}
