package vm

import "context"

// LuaLoadLibProvider controls package.loadlib() behavior.
//
// Standard C Lua modules (.so/.dll) cannot be loaded directly — they are
// compiled against the PUC-Rio C API (lua_State*, lua_push*, etc.) which has
// no equivalent in GoLua. This provider exists so the host application can
// implement its own native module loading strategy, for example exposing
// Go-implemented bindings, using cgo to bridge platform-specific libraries,
// or mapping module names to pre-registered native functions.
//
// When unset, stdlib.Open still exposes package.loadlib but it returns
// Lua's standard "absent" failure triple.
type LuaLoadLibProvider interface {
	// LoadLib resolves (path, init) and returns a callable loader function.
	//
	// path and init are the two package.loadlib arguments (matching the
	// Lua 5.4 signature). The host decides how to interpret them — they
	// need not refer to actual shared-object files.
	//
	// caller provides context about the Lua call site (source, line).
	// On success, return a NativeFunc with empty errmsg/where. On failure,
	// return a nil loader plus an error message and one of Lua's failure tags:
	// "open", "init", or "absent".
	LoadLib(ctx context.Context, path, init string, caller *LuaCallerContext) (loader NativeFunc, errmsg string, where string)
}
