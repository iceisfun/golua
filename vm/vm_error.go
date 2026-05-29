package vm

import (
	"fmt"
	"strings"
)

// runtimeError creates a formatted error prefixed with the current source location (source:line:).
func (vm *VM) runtimeError(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if len(vm.callStack) > 0 {
		frame := &vm.callStack[len(vm.callStack)-1]
		if frame.closure != nil {
			proto := frame.closure.Proto
			pc := frame.pc - 1
			if pc >= 0 && pc < len(proto.Lines) {
				return fmt.Errorf("%s:%d: %s", shortSrc(proto.Source), proto.Lines[pc], msg)
			}
			if proto.Source != "" {
				// Stripped functions have no line info; use -1 like Lua 5.4.
				return fmt.Errorf("%s:-1: %s", shortSrc(proto.Source), msg)
			}
		}
	}
	return fmt.Errorf("%s", msg)
}

// callerLocationPrefix returns the "source:line: " prefix for the Lua frame
// that called the current (top-of-stack) native function, mirroring Lua's
// luaL_where(L, 1). Returns "" when there is no such caller, the caller is
// itself a native frame, or the caller has no source info.
func (vm *VM) callerLocationPrefix() string {
	// The top frame is the native function that panicked; the frame below it
	// is its caller. If that caller is also native (e.g., pcall calling type),
	// luaL_where adds nothing.
	n := len(vm.callStack)
	if n < 2 {
		return ""
	}
	callerFrame := &vm.callStack[n-2]
	if callerFrame.closure == nil {
		return ""
	}
	proto := callerFrame.closure.Proto
	pc := callerFrame.pc - 1
	if pc >= 0 && pc < len(proto.Lines) {
		return fmt.Sprintf("%s:%d: ", shortSrc(proto.Source), proto.Lines[pc])
	}
	if proto.Source != "" {
		// Stripped functions have no line info; use -1 like Lua 5.4.
		return fmt.Sprintf("%s:-1: ", shortSrc(proto.Source))
	}
	return ""
}

// AddCallerLocation prepends the calling Lua frame's source:line: prefix
// to a plain error message from a native function. This mirrors Lua 5.4's
// luaG_addinfo / luaL_where(L, 1) which adds the caller location to stdlib errors.
// Only adds the prefix if the immediate caller of the erroring native function
// is a Lua frame (not another native function like pcall). If the message
// already carries that exact prefix it is not duplicated — guarding against
// re-prefixing an error that was already positioned at its origin.
func (vm *VM) AddCallerLocation(msg string) string {
	prefix := vm.callerLocationPrefix()
	if prefix == "" || strings.HasPrefix(msg, prefix) {
		return msg
	}
	return prefix + msg
}

// PrependCallerLocation unconditionally prepends the caller's source:line:
// prefix (luaL_where(L, 1)) to msg, even when msg already begins with that
// exact prefix. coroutine.wrap re-raises string errors this way: Lua's
// luaB_auxwrap concatenates luaL_where(L, 1) onto the propagated error
// unconditionally, so an error whose origin line matches the wrap call site
// legitimately shows the prefix twice.
func (vm *VM) PrependCallerLocation(msg string) string {
	return vm.callerLocationPrefix() + msg
}

// CallerFuncName inspects the calling Lua frame's bytecode to determine the
// name under which the current native function was called. This mirrors
// Lua 5.4's luaG_funcnamefromcode behavior. Returns (name, nameWhat) where
// nameWhat is "global", "field", "method", "local", "upvalue", or "".
// Returns ("", "") if the name cannot be determined.
func (vm *VM) CallerFuncName() (name, nameWhat string) {
	n := len(vm.callStack)
	if n < 2 {
		return "", ""
	}
	// The top frame is the native function. The frame below is the Lua caller.
	callerFrame := &vm.callStack[n-2]
	if callerFrame.closure == nil {
		return "", ""
	}
	return vm.funcNameFromCall(callerFrame)
}

// ArgErrorFuncName resolves the name of the current native function for use
// in argument error messages, matching Lua 5.4's luaL_argerror behavior.
// It first tries bytecode resolution (CallerFuncName). If that fails (e.g.,
// when called via pcall), it falls back to searching globals for the function
// value (pushglobalfuncname equivalent). Returns ("?", "") if neither works.
func (vm *VM) ArgErrorFuncName() (name, nameWhat string) {
	name, nameWhat = vm.CallerFuncName()
	if name != "" {
		return name, nameWhat
	}
	// Bytecode resolution failed — try global table lookup.
	n := len(vm.callStack)
	if n > 0 {
		fn := vm.callStack[n-1].funcValue
		if resolved, ok := vm.lookupNativeFuncName(fn); ok {
			return resolved, ""
		}
	}
	return "?", ""
}

// ObjTypeName returns the type name for a value, checking __name in
// the metatable first (matching Lua 5.4's luaT_objtypename).
func (vm *VM) ObjTypeName(v Value) string {
	name := vm.getMetafield(v, "__name")
	if name.IsString() {
		return name.AsString()
	}
	return v.Type()
}

// varInfo returns a parenthesized context string like " (global 'bbbb')" for
// the value in the given register at the current PC, by inspecting the bytecode.
// Returns "" if the name cannot be determined. This mirrors Lua 5.4's varinfo()
// in ldebug.c.
func (vm *VM) varInfo(reg int) string {
	if len(vm.callStack) > 0 {
		frame := &vm.callStack[len(vm.callStack)-1]
		if frame.closure != nil {
			proto := frame.closure.Proto
			pc := frame.pc - 1
			name, what := regObjName(proto, pc, reg)
			if name != "" {
				return fmt.Sprintf(" (%s '%s')", what, name)
			}
		}
	}
	return ""
}

// runtimeErrorForNumber creates a "number has no integer representation" error
// that includes the name of the offending value (e.g., "field 'huge'")
// when it can be determined from the bytecode, matching Lua 5.4's luaG_forerror.
func (vm *VM) runtimeErrorForNumber(reg int) error {
	if len(vm.callStack) > 0 {
		frame := &vm.callStack[len(vm.callStack)-1]
		if frame.closure != nil {
			proto := frame.closure.Proto
			pc := frame.pc - 1
			name, what := regObjName(proto, pc, reg)
			if name != "" {
				return vm.runtimeError("number (%s '%s') has no integer representation", what, name)
			}
		}
	}
	return vm.runtimeError("number has no integer representation")
}
