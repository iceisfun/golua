package vm

// LuaDebugCaps declares which debug-library operations are exposed to Lua.
type LuaDebugCaps struct {
	// AllowTraceback enables debug.traceback.
	AllowTraceback bool
	// AllowStackDepth enables debug.stackdepth.
	AllowStackDepth bool
	// AllowWhere enables debug.where.
	AllowWhere bool
	// AllowGetInfo enables debug.getinfo.
	AllowGetInfo bool
	// AllowGetUpvalue enables debug.getupvalue.
	AllowGetUpvalue bool
	// AllowSetUpvalue enables debug.setupvalue.
	AllowSetUpvalue bool
	// AllowUpvalueID enables debug.upvalueid.
	AllowUpvalueID bool
	// AllowGetLocal enables debug.getlocal.
	AllowGetLocal bool
	// AllowSetLocal enables debug.setlocal.
	AllowSetLocal bool
	// AllowGetRegistry enables debug.getregistry.
	AllowGetRegistry bool
	// AllowGetMetatable enables debug.getmetatable.
	AllowGetMetatable bool
	// AllowSetMetatable enables debug.setmetatable.
	AllowSetMetatable bool
	// AllowSetHook enables debug.sethook.
	AllowSetHook bool
	// AllowGetHook enables debug.gethook.
	AllowGetHook bool
	// AllowUpvalueJoin enables debug.upvaluejoin.
	AllowUpvalueJoin bool
	// AllowSetCStackLimit enables debug.setcstacklimit compatibility behavior.
	AllowSetCStackLimit bool
	// AllowGetUserValue enables debug.getuservalue.
	AllowGetUserValue bool
	// AllowSetUserValue enables debug.setuservalue.
	AllowSetUserValue bool
}

// LuaDebugProvider is a capability interface for exposing the Lua debug library.
// Without a provider, the debug table is absent from the VM.
type LuaDebugProvider interface {
	// Capabilities declares which diagnostic behaviors are allowed.
	Capabilities() LuaDebugCaps
}

// DefaultDebugProvider enables all diagnostic functions.
type DefaultDebugProvider struct{}

// NewDefaultDebugProvider creates a debug provider with all diagnostics enabled.
func NewDefaultDebugProvider() *DefaultDebugProvider {
	return &DefaultDebugProvider{}
}

// Capabilities returns caps with all diagnostic functions enabled.
func (p *DefaultDebugProvider) Capabilities() LuaDebugCaps {
	return LuaDebugCaps{
		AllowTraceback:      true,
		AllowStackDepth:     true,
		AllowWhere:          true,
		AllowGetInfo:        true,
		AllowGetUpvalue:     true,
		AllowSetUpvalue:     true,
		AllowUpvalueID:      true,
		AllowGetLocal:       true,
		AllowSetLocal:       true,
		AllowGetRegistry:    true,
		AllowGetMetatable:   true,
		AllowSetMetatable:   true,
		AllowSetHook:        true,
		AllowGetHook:        true,
		AllowUpvalueJoin:    true,
		AllowSetCStackLimit: true,
		AllowGetUserValue:   true,
		AllowSetUserValue:   true,
	}
}
