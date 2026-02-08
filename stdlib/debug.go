package stdlib

import "github.com/iceisfun/golua/vm"

// openDebug registers the diagnostic debug table if a DebugProvider is set.
func openDebug(v *vm.VM) {
	provider := v.DebugProvider()
	if provider == nil {
		return
	}

	caps := provider.Capabilities()
	debug := vm.NewEmptyTable()

	if caps.AllowTraceback {
		debug.SetString("traceback", vm.NewNativeFunc(luaDebugTraceback))
	}

	if caps.AllowStackDepth {
		debug.SetString("stackdepth", vm.NewNativeFunc(luaDebugStackDepth))
	}

	if caps.AllowWhere {
		debug.SetString("where", vm.NewNativeFunc(luaDebugWhere))
	}

	v.SetGlobal("debug", vm.NewTable(debug))
}

// debug.traceback([message [, level]])
func luaDebugTraceback(v *vm.VM) int {
	msg := ""
	if !v.Get(1).IsNil() {
		msg = valueToString(v.Get(1))
	}

	level := 1
	if !v.Get(2).IsNil() {
		if l, ok := v.Get(2).ToInt(); ok {
			level = int(l)
		}
	}

	v.Set(0, vm.NewString(v.Traceback(msg, level)))
	return 1
}

// debug.stackdepth()
func luaDebugStackDepth(v *vm.VM) int {
	v.Set(0, vm.NewInt(int64(v.StackDepth())))
	return 1
}

// debug.where([level])
func luaDebugWhere(v *vm.VM) int {
	level := 1
	if !v.Get(1).IsNil() {
		if l, ok := v.Get(1).ToInt(); ok {
			level = int(l)
		}
	}

	source, line, ok := v.Where(level)
	if !ok {
		v.Set(0, vm.Nil)
		return 1
	}

	v.Set(0, vm.NewString(source))
	v.Set(1, vm.NewInt(int64(line)))
	return 2
}
