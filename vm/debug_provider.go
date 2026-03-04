package vm

// LuaDebugCaps declares which diagnostic debug operations are allowed.
type LuaDebugCaps struct {
	AllowTraceback  bool
	AllowStackDepth bool
	AllowWhere      bool
	AllowGetInfo     bool
	AllowGetUpvalue  bool
	AllowSetUpvalue  bool
	AllowUpvalueID   bool
	AllowGetLocal    bool
	AllowSetLocal    bool
	AllowGetRegistry bool
	AllowGetMetatable bool
	AllowSetMetatable bool
	AllowSetHook      bool
	AllowGetHook      bool
	AllowUpvalueJoin  bool
}

// LuaDebugProvider is a capability interface for diagnostic debug operations.
// This is NOT the standard Lua debug library. It exposes only read-only
// diagnostic functions (traceback, stack depth, source location). No hooks,
// no local/upvalue mutation, no bytecode inspection.
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
		AllowTraceback:  true,
		AllowStackDepth: true,
		AllowWhere:      true,
		AllowGetInfo:     true,
		AllowGetUpvalue:  true,
		AllowSetUpvalue:  true,
		AllowUpvalueID:   true,
		AllowGetLocal:    true,
		AllowSetLocal:    true,
		AllowGetRegistry: true,
		AllowGetMetatable: true,
		AllowSetMetatable: true,
		AllowSetHook:      true,
		AllowGetHook:      true,
		AllowUpvalueJoin:  true,
	}
}
