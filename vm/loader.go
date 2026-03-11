package vm

// LuaCallerContext describes the Lua frame that requested a chunk load.
type LuaCallerContext struct {
	// ScriptName is the display name of the currently executing chunk.
	ScriptName string
	// VMID is an optional host-defined VM identifier for audit or policy checks.
	VMID string
	// CallDepth is the current Lua call depth when the load request is made.
	CallDepth int
}

// LuaLoaderCaps declares which optional loading helpers are exposed to Lua.
type LuaLoaderCaps struct {
	// AllowDofile enables the global dofile helper.
	AllowDofile bool
	// AllowLoadfile enables the global loadfile helper.
	AllowLoadfile bool
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
