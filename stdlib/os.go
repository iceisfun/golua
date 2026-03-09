package stdlib

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/iceisfun/golua/vm"
)

// formatPathError formats an OS error for Lua, stripping the Go operation prefix.
// Go returns "remove /path: error" or "rename /old /new: error".
// Lua expects "/path: Error description" with errno.
func formatPathError(name string, err error) (string, int) {
	var errno int
	var errMsg string

	if pathErr, ok := err.(*os.PathError); ok {
		if sysErr, ok := pathErr.Err.(syscall.Errno); ok {
			errno = int(sysErr)
		} else {
			errno = extractErrnoFromError(err)
		}
		// Capitalize the first letter of the error description
		errMsg = capitalizeError(pathErr.Err.Error())
	} else if linkErr, ok := err.(*os.LinkError); ok {
		if sysErr, ok := linkErr.Err.(syscall.Errno); ok {
			errno = int(sysErr)
		} else {
			errno = extractErrnoFromError(err)
		}
		errMsg = capitalizeError(linkErr.Err.Error())
	} else {
		errno = extractErrnoFromError(err)
		errMsg = err.Error()
	}

	return fmt.Sprintf("%s: %s", name, errMsg), errno
}

// capitalizeError capitalizes the first letter of an error string to match Lua format.
func capitalizeError(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// extractErrnoFromError extracts an errno from an error chain.
func extractErrnoFromError(err error) int {
	for err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			return int(errno)
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return 2 // ENOENT as default
}

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

		// Optional fields with Lua 5.4 defaults: hour=12, min=0, sec=0
		defaults := map[string]int{"hour": 12, "min": 0, "sec": 0}
		for _, key := range []string{"hour", "min", "sec"} {
			val := t.Get(vm.NewString(key))
			if !val.IsNil() {
				i, ok := val.ToInt()
				if !ok {
					panic(fmt.Sprintf("field '%s' is not an integer", key))
				}
				dateTable[key] = int(i)
			} else {
				dateTable[key] = defaults[key]
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
			arg1 := v.Get(1)
			if arg1.IsString() {
				format = arg1.AsString()
			} else if arg1.IsNumber() {
				// Lua coerces numbers to strings for string arguments
				format = valueToString(arg1)
			} else {
				callerArgError(v, 1, "os.date", fmt.Sprintf("string expected, got %s", arg1.Type()))
			}
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
			panic(fmt.Sprintf("bad argument #1 to 'os.date' (%s)", err.Error()))
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
		if v.ArgCount() < 1 || name.IsNil() {
			callerArgError(v, 1, "os.remove", "string expected, got no value")
		}
		if !name.IsString() && !name.IsNumber() {
			callerArgError(v, 1, "os.remove", fmt.Sprintf("string expected, got %s", name.Type()))
		}
		nameStr := name.AsString()
		err := ioProvider.Remove(nameStr)
		if err != nil {
			msg, errno := formatPathError(nameStr, err)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(msg))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
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
		arg1 := v.Get(1)
		if v.ArgCount() < 1 || arg1.IsNil() {
			callerArgError(v, 1, "os.rename", "string expected, got no value")
		}
		if !arg1.IsString() && !arg1.IsNumber() {
			callerArgError(v, 1, "os.rename", fmt.Sprintf("string expected, got %s", arg1.Type()))
		}
		arg2 := v.Get(2)
		if v.ArgCount() < 2 || arg2.IsNil() {
			callerArgError(v, 2, "os.rename", "string expected, got no value")
		}
		if !arg2.IsString() && !arg2.IsNumber() {
			callerArgError(v, 2, "os.rename", fmt.Sprintf("string expected, got %s", arg2.Type()))
		}
		oldname := arg1.AsString()
		newname := arg2.AsString()
		err := ioProvider.Rename(oldname, newname)
		if err != nil {
			msg, errno := formatPathError(oldname, err)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(msg))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
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
		if v.ArgCount() < 1 || name.IsNil() {
			callerArgError(v, 1, "os.getenv", "string expected, got no value")
		}
		if !name.IsString() && !name.IsNumber() {
			callerArgError(v, 1, "os.getenv", fmt.Sprintf("string expected, got %s", name.Type()))
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
