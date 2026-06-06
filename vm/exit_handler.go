package vm

import (
	"context"
	"fmt"
)

// LuaExitHandler is a capability interface for handling os.exit calls.
// Implementations control what happens when Lua code calls os.exit.
type LuaExitHandler interface {
	// Exit handles an os.exit call with the given exit code.
	// If close is true, to-be-closed variables should be closed first.
	Exit(ctx context.Context, code int, close bool)
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

// CoroutineSelfClose is the sentinel panic value used by Lua 5.5's
// coroutine.close(coroutine.running()): the running coroutine longjumps
// out cleanly, returning (true) to its resumer. The stdlib coroutine
// runner detects this sentinel in its recover handler and treats it as
// a normal termination, not an error. coroutine.close runs any pending
// <close> handlers before panicking with this value.  ProtectedCall
// propagates the sentinel through pcall/xpcall boundaries (just like
// LuaExitError), since the long-jump terminates the entire coroutine.
//
// If a pending <close> handler errors while the running coroutine is being
// self-closed, the coroutine still terminates via this sentinel, but the
// handler's error is carried in Err so the resumer observes (false, errval)
// instead of (true). Err is nil for the normal (no-error) self-close.
type CoroutineSelfClose struct {
	Err    error
	HasErr bool
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
func (h *DefaultExitHandler) Exit(ctx context.Context, code int, close bool) {
	panic(&LuaExitError{Code: code})
}
