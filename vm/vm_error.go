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
				return fmt.Errorf("%s:%d: %s", proto.Source, proto.Lines[pc], msg)
			}
			if proto.Source != "" {
				return fmt.Errorf("%s: %s", proto.Source, msg)
			}
		}
	}
	return fmt.Errorf("%s", msg)
}
