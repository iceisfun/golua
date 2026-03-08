package stdlib

import (
	"fmt"
	"time"

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

	// tmpname and remove route through the IO provider
	ioProvider := v.IoProvider()
	if ioProvider != nil {
		if caps.AllowTmpName {
			osTable.SetString("tmpname", vm.NewNativeFunc(makeOsTmpname(ioProvider)))
		}
		if caps.AllowRemove {
			osTable.SetString("remove", vm.NewNativeFunc(makeOsRemove(ioProvider)))
		}
		if caps.AllowRename {
			osTable.SetString("rename", vm.NewNativeFunc(makeOsRename(ioProvider)))
		}
	}

	// setlocale is always available (stub, returns "C" only)
	osTable.SetString("setlocale", vm.NewNativeFunc(osSetlocale))

	// execute routes through the exec provider
	execProvider := v.ExecProvider()
	if execProvider != nil && caps.AllowExecute {
		osTable.SetString("execute", vm.NewNativeFunc(makeOsExecute(execProvider)))
	}

	// exit routes through the exit handler
	exitHandler := v.ExitHandler()
	if exitHandler != nil && caps.AllowExit {
		osTable.SetString("exit", vm.NewNativeFunc(makeOsExit(v, exitHandler)))
	}

	v.SetGlobal("os", vm.NewTable(osTable))
}

// makeOsExecute creates the os.execute([command]) function.
func makeOsExecute(provider vm.LuaExecProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		if v.ArgCount() < 1 || v.Get(1).IsNil() {
			// No argument or nil: check if shell is available
			v.Set(0, vm.True)
			return 1
		}
		cmd := v.Get(1)
		if !cmd.IsString() {
			callerArgError(v, 1, "os.execute", fmt.Sprintf("string expected, got %s", cmd.Type()))
		}
		ok, exitType, exitCode := provider.Execute(cmd.AsString())
		if ok {
			v.Set(0, vm.True)
		} else {
			v.Set(0, vm.Nil)
		}
		v.Set(1, vm.NewString(exitType))
		v.Set(2, vm.NewInt(int64(exitCode)))
		return 3
	}
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
			panic("bad argument #1 to 'os.time' (table expected)")
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

		// Normalize fields back into the table (like C's mktime)
		norm := time.Unix(ts, 0).Local()
		t.Set(vm.NewString("year"), vm.NewInt(int64(norm.Year())))
		t.Set(vm.NewString("month"), vm.NewInt(int64(norm.Month())))
		t.Set(vm.NewString("day"), vm.NewInt(int64(norm.Day())))
		t.Set(vm.NewString("hour"), vm.NewInt(int64(norm.Hour())))
		t.Set(vm.NewString("min"), vm.NewInt(int64(norm.Minute())))
		t.Set(vm.NewString("sec"), vm.NewInt(int64(norm.Second())))
		t.Set(vm.NewString("yday"), vm.NewInt(int64(norm.YearDay())))
		wday := int(norm.Weekday()) + 1 // Lua: Sunday=1
		t.Set(vm.NewString("wday"), vm.NewInt(int64(wday)))

		v.Set(0, vm.NewInt(ts))
		return 1
	}
}

// osDifftime implements os.difftime(t2, t1).
func osDifftime(v *vm.VM) int {
	t2, ok2 := v.Get(1).ToNumber()
	t1, ok1 := v.Get(2).ToNumber()
	if !ok1 || !ok2 {
		panic("bad argument to 'os.difftime' (number expected)")
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
				panic("bad argument #2 to 'os.date' (number expected)")
			}
			timestamp = ts
		} else {
			ts, _ := provider.Time(nil)
			timestamp = ts
		}

		// Check for "*t" format — return a table
		checkFmt := format
		utc := false
		if len(checkFmt) > 0 && checkFmt[0] == '!' {
			utc = true
			checkFmt = checkFmt[1:]
		}

		if checkFmt == "*t" {
			dt := provider.DateTable(timestamp, utc)
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

// makeOsTmpname creates the os.tmpname() function.
func makeOsTmpname(ioProvider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name, err := ioProvider.TmpName()
		if err != nil {
			panic(fmt.Sprintf("unable to generate a unique filename: %s", err.Error()))
		}
		v.Set(0, vm.NewString(name))
		return 1
	}
}

// makeOsRemove creates the os.remove(filename) function.
func makeOsRemove(ioProvider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1)
		if name.IsNil() {
			panic("bad argument #1 to 'os.remove' (string expected, got nil)")
		}
		err := ioProvider.Remove(name.AsString())
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
		v.Set(0, vm.True)
		return 1
	}
}

// osSetlocale implements os.setlocale([locale [, category]]).
// This is a stub that only supports the "C" locale (Go has no locale support).
// Returns "C" for nil/empty/"C" queries, nil for anything else.
func osSetlocale(v *vm.VM) int {
	locale := ""
	if v.ArgCount() >= 1 && !v.Get(1).IsNil() {
		locale = v.Get(1).AsString()
	}

	// Query (nil or empty string) or set to "C" — always succeeds
	if locale == "" || locale == "C" {
		v.Set(0, vm.NewString("C"))
		return 1
	}

	// Any other locale is not supported
	v.Set(0, vm.Nil)
	return 1
}

// makeOsRename creates the os.rename(oldname, newname) function.
func makeOsRename(ioProvider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		if v.ArgCount() < 1 || v.Get(1).IsNil() {
			callerArgError(v, 1, "os.rename", "string expected, got nil")
		}
		if v.ArgCount() < 2 || v.Get(2).IsNil() {
			callerArgError(v, 2, "os.rename", "string expected, got nil")
		}
		oldname := v.Get(1).AsString()
		newname := v.Get(2).AsString()
		err := ioProvider.Rename(oldname, newname)
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
		v.Set(0, vm.True)
		return 1
	}
}

// makeOsExit creates the os.exit([code [, close]]) function.
// code defaults to true (=0). false means exit code 1.
// If close is true, to-be-closed variables are closed first.
func makeOsExit(vmRef *vm.VM, handler vm.LuaExitHandler) vm.NativeFunc {
	return func(v *vm.VM) int {
		code := 0
		arg1 := v.Get(1)
		if !arg1.IsNil() {
			if arg1.IsBool() {
				if arg1.AsBool() {
					code = 0
				} else {
					code = 1
				}
			} else {
				i, ok := arg1.ToInt()
				if !ok {
					callerArgError(v, 1, "os.exit", fmt.Sprintf("number expected, got %s", arg1.Type()))
				}
				code = int(i)
			}
		}

		closeFlag := false
		arg2 := v.Get(2)
		if !arg2.IsNil() {
			closeFlag = arg2.ToBool()
		}

		if closeFlag {
			// Close all to-be-closed variables
			v.CloseAllTBC()
		}

		handler.Exit(code, closeFlag)
		return 0 // unreachable
	}
}

// makeOsGetenv creates the os.getenv(varname) function.
func makeOsGetenv(provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1)
		if name.IsNil() {
			panic("bad argument #1 to 'os.getenv' (string expected)")
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
