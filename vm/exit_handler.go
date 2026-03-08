package vm

import "fmt"

// LuaExitHandler is a capability interface for handling os.exit calls.
// Implementations control what happens when Lua code calls os.exit.
type LuaExitHandler interface {
	// Exit handles an os.exit call with the given exit code.
	// If close is true, to-be-closed variables should be closed first.
	Exit(code int, close bool)
}

// LuaExitError is a sentinel error used by DefaultExitHandler to stop VM
// execution. It is recognized by ProtectedCall and propagated without
// being caught by pcall/xpcall.
type LuaExitError struct {
	Code int
}

func (e *LuaExitError) Error() string {
	return fmt.Sprintf("os.exit(%d)", e.Code)
}

// DefaultExitHandler stops VM execution by panicking with a LuaExitError
// sentinel that propagates through ProtectedCall boundaries.
type DefaultExitHandler struct{}

// NewDefaultExitHandler creates a new DefaultExitHandler.
func NewDefaultExitHandler() *DefaultExitHandler {
	return &DefaultExitHandler{}
}

// Exit panics with LuaExitError to stop VM execution.
// The close parameter is handled by os.exit before calling this handler.
func (h *DefaultExitHandler) Exit(code int, close bool) {
	panic(&LuaExitError{Code: code})
}
