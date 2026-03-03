package stdlib

import (
	"fmt"

	"github.com/iceisfun/golua/vm"
)

// openOs registers the os library if an OsProvider is set.
func openOs(v *vm.VM) {
	provider := v.OsProvider()
	if provider == nil {
		return
	}

	osTable := vm.NewEmptyTable()
	caps := provider.Capabilities()

	if caps.AllowTime {
		osTable.SetString("clock", vm.NewNativeFunc(makeOsClock(provider)))
		osTable.SetString("time", vm.NewNativeFunc(makeOsTime(v, provider)))
		osTable.SetString("difftime", vm.NewNativeFunc(osDifftime))
	}

	if caps.AllowDate {
		osTable.SetString("date", vm.NewNativeFunc(makeOsDate(v, provider)))
	}

	if caps.AllowGetenv {
		osTable.SetString("getenv", vm.NewNativeFunc(makeOsGetenv(provider)))
	}

	v.SetGlobal("os", vm.NewTable(osTable))
}

// makeOsClock creates the os.clock() function.
func makeOsClock(provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		v.Set(0, vm.NewFloat(provider.Clock()))
		return 1
	}
}

// makeOsTime creates the os.time([table]) function.
func makeOsTime(vmRef *vm.VM, provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		arg := v.Get(1)
		if arg.IsNil() {
			ts, err := provider.Time(nil)
			if err != nil {
				panic(err.Error())
			}
			v.Set(0, vm.NewInt(ts))
			return 1
		}

		if !arg.IsTable() {
			panic("bad argument #1 to 'time' (table expected)")
		}

		t := arg.AsTable()
		dateTable := make(map[string]int)

		// Required fields
		for _, key := range []string{"year", "month", "day"} {
			val := t.Get(vm.NewString(key))
			if val.IsNil() {
				panic(fmt.Sprintf("field '%s' missing in date table", key))
			}
			i, ok := val.ToInt()
			if !ok {
				panic(fmt.Sprintf("field '%s' is not an integer", key))
			}
			dateTable[key] = int(i)
		}

		// Optional fields
		for _, key := range []string{"hour", "min", "sec"} {
			val := t.Get(vm.NewString(key))
			if !val.IsNil() {
				i, ok := val.ToInt()
				if !ok {
					panic(fmt.Sprintf("field '%s' is not an integer", key))
				}
				dateTable[key] = int(i)
			}
		}

		ts, err := provider.Time(dateTable)
		if err != nil {
			panic(err.Error())
		}
		v.Set(0, vm.NewInt(ts))
		return 1
	}
}

// osDifftime implements os.difftime(t2, t1).
func osDifftime(v *vm.VM) int {
	t2, ok2 := v.Get(1).ToNumber()
	t1, ok1 := v.Get(2).ToNumber()
	if !ok1 || !ok2 {
		panic("bad argument to 'difftime' (number expected)")
	}
	v.Set(0, vm.NewFloat(t2-t1))
	return 1
}

// makeOsDate creates the os.date([format [, time]]) function.
func makeOsDate(vmRef *vm.VM, provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		format := "%c"
		if !v.Get(1).IsNil() {
			format = v.Get(1).AsString()
		}

		var timestamp int64
		if !v.Get(2).IsNil() {
			ts, ok := v.Get(2).ToInt()
			if !ok {
				panic("bad argument #2 to 'date' (number expected)")
			}
			timestamp = ts
		} else {
			ts, _ := provider.Time(nil)
			timestamp = ts
		}

		// Check for "*t" format — return a table
		checkFmt := format
		if len(checkFmt) > 0 && checkFmt[0] == '!' {
			checkFmt = checkFmt[1:]
		}

		if checkFmt == "*t" {
			dt := provider.DateTable(timestamp)
			t := vm.NewEmptyTable()
			t.SetString("year", vm.NewInt(int64(dt["year"])))
			t.SetString("month", vm.NewInt(int64(dt["month"])))
			t.SetString("day", vm.NewInt(int64(dt["day"])))
			t.SetString("hour", vm.NewInt(int64(dt["hour"])))
			t.SetString("min", vm.NewInt(int64(dt["min"])))
			t.SetString("sec", vm.NewInt(int64(dt["sec"])))
			t.SetString("wday", vm.NewInt(int64(dt["wday"])))
			t.SetString("yday", vm.NewInt(int64(dt["yday"])))
			isdst := false
			if dt["isdst"] != 0 {
				isdst = true
			}
			t.SetString("isdst", vm.NewBool(isdst))
			v.Set(0, vm.NewTable(t))
			return 1
		}

		result, err := provider.Date(format, timestamp)
		if err != nil {
			panic(err.Error())
		}
		v.Set(0, vm.NewString(result))
		return 1
	}
}

// makeOsGetenv creates the os.getenv(varname) function.
func makeOsGetenv(provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1)
		if name.IsNil() {
			panic("bad argument #1 to 'getenv' (string expected)")
		}

		val, ok := provider.Getenv(name.AsString())
		if !ok {
			v.Set(0, vm.Nil)
			return 1
		}
		v.Set(0, vm.NewString(val))
		return 1
	}
}
