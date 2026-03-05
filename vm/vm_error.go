package vm

import "fmt"

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

// ObjTypeName returns the type name for a value, checking __name in
// the metatable first (matching Lua 5.4's luaT_objtypename).
func (vm *VM) ObjTypeName(v Value) string {
	name := vm.getMetafield(v, "__name")
	if name.IsString() {
		return name.AsString()
	}
	return v.Type()
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

