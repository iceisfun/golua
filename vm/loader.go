package vm

// LuaCallerContext provides context about who is requesting a chunk load.
type LuaCallerContext struct {
	ScriptName string // current executing chunk
	VMID       string // optional VM identifier
	CallDepth  int    // how deep the call stack is
}

// LuaLoaderCaps declares which optional loading behaviors are allowed.
type LuaLoaderCaps struct {
	AllowDofile   bool // execute immediately helper
	AllowLoadfile bool // filesystem-style semantics (still routed!)
}

// LuaCodeProvider is a capability interface for sandboxed code loading.
// Implementations control what Lua source code is available and from where,
// enabling secure embedding without granting filesystem access.
//
// The VM calls LoadChunk when Lua code invokes load(), loadfile(), or dofile().
// Returning an error prevents loading; the error message is propagated to Lua.
type LuaCodeProvider interface {
	// LoadChunk resolves a Lua chunk and returns its source code.
	//
	// name:
	//   Logical name requested by Lua (e.g. "init", "tests/math", "foo.lua")
	//
	// caller:
	//   Optional context about *who* is asking (script name, VM id, etc)
	//
	// Returns:
	//   - source code (text only)
	//   - a display name for stack traces / debug
	//   - error if not permitted or not found
	//
	LoadChunk(name string, caller *LuaCallerContext) (source []byte, chunkName string, err error)

	// Capabilities declares which optional behaviors are allowed.
	// This lets Lua expose or hide helpers safely.
	Capabilities() LuaLoaderCaps
}
