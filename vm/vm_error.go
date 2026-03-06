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
				return fmt.Errorf("%s: %s", shortSrc(proto.Source), msg)
			}
		}
	}
	return fmt.Errorf("%s", msg)
}

// addCallerLocation prepends the calling Lua frame's source:line: prefix
// to a plain error message from a native function. This mirrors Lua 5.4's
// luaG_addinfo / luaL_where which adds the caller location to stdlib errors.
// It walks the call stack to find the Lua frame that called the native function.
// Returns the original message if no Lua frame is found or if the message
// already has a location prefix.
func (vm *VM) addCallerLocation(msg string) string {
	// Walk the call stack backwards to find the first Lua frame
	for i := len(vm.callStack) - 1; i >= 0; i-- {
		frame := &vm.callStack[i]
		if frame.closure != nil {
			proto := frame.closure.Proto
			pc := frame.pc - 1
			if pc >= 0 && pc < len(proto.Lines) {
				prefix := fmt.Sprintf("%s:%d: ", shortSrc(proto.Source), proto.Lines[pc])
				if strings.HasPrefix(msg, prefix) {
					return msg // already has this prefix
				}
				return prefix + msg
			}
			if proto.Source != "" {
				prefix := shortSrc(proto.Source) + ": "
				if strings.HasPrefix(msg, prefix) {
					return msg
				}
				return prefix + msg
			}
			break
		}
	}
	return msg
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

