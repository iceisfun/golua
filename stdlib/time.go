package stdlib

import "github.com/iceisfun/golua/vm"

// openTime registers the time library if a TimeProvider is set.
func openTime(v *vm.VM) {
	provider := v.TimeProvider()
	if provider == nil {
		return
	}

	timeTable := vm.NewEmptyTable()
	timeTable.SetString("now", vm.NewNativeFunc(makeTimeNow(provider)))
	timeTable.SetString("since", vm.NewNativeFunc(makeTimeSince(provider)))
	v.SetGlobal("time", vm.NewTable(timeTable))
}

// makeTimeNow creates the time.now() function.
// Returns current time in milliseconds.
func makeTimeNow(provider vm.LuaTimeProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		v.Set(0, vm.NewInt(provider.Now()))
		return 1
	}
}

// makeTimeSince creates the time.since(t) function.
// Returns milliseconds elapsed since t.
func makeTimeSince(provider vm.LuaTimeProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		t := getInt(v, 1, "time.since")
		v.Set(0, vm.NewInt(provider.Now()-t))
		return 1
	}
}
