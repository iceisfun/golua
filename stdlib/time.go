package stdlib

import "github.com/iceisfun/golua/v2/vm"

// openTime registers the time library if a TimeProvider is set.
func openTime(v *vm.VM) {
	provider := v.TimeProvider()
	if provider == nil {
		return
	}

	timeTable := vm.NewEmptyTable()
	timeTable.SetString("now", vm.NewNativeFunc(makeTimeNow(provider)))
	timeTable.SetString("since", vm.NewNativeFunc(makeTimeSince(provider)))
	timeTable.SetString("tick", vm.NewNativeFunc(makeTimeTick(provider)))
	timeTable.SetString("once", vm.NewNativeFunc(makeTimeOnce(provider)))
	v.SetGlobal("time", vm.NewTable(timeTable))
}

// makeTimeNow creates the time.now() function.
// Returns current time in milliseconds.
func makeTimeNow(provider vm.LuaTimeProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		v.Set(0, vm.NewInt(provider.Now(v.Context())))
		return 1
	}
}

// makeTimeSince creates the time.since(t) function.
// Returns milliseconds elapsed since t.
func makeTimeSince(provider vm.LuaTimeProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		t := getInt(v, 1, "time.since")
		v.Set(0, vm.NewInt(provider.Now(v.Context())-t))
		return 1
	}
}

// makeTimeTick creates the time.tick([name,] ms) function.
// Returns true once per ms interval, false otherwise.
// If name is omitted, the callsite (source:line) is used as the key.
func makeTimeTick(provider vm.LuaTimeProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		arg1 := v.Get(1)
		var key string
		var ms int64

		if arg1.IsString() {
			// time.tick("name", ms)
			key = arg1.AsString()
			ms = getInt(v, 2, "time.tick")
		} else {
			// time.tick(ms) — use callsite as key
			ms = getInt(v, 1, "time.tick")
			key = v.GetSourceLocation(1)
			if key == "" {
				key = "tick"
			}
		}

		v.Set(0, vm.NewBool(provider.Tick(v.Context(), key, ms)))
		return 1
	}
}

// makeTimeOnce creates the time.once([name]) function.
// Returns true on the first call for a given key, false on all subsequent calls.
// If name is omitted, the callsite (source:line) is used as the key.
func makeTimeOnce(provider vm.LuaTimeProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		var key string

		arg1 := v.Get(1)
		if arg1.IsString() {
			// time.once("name")
			key = arg1.AsString()
		} else {
			// time.once() — use callsite as key
			key = v.GetSourceLocation(1)
			if key == "" {
				key = "once"
			}
		}

		v.Set(0, vm.NewBool(provider.Once(v.Context(), key)))
		return 1
	}
}
