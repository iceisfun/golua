package stdlib

import (
	"fmt"
	"math"
	"os"
	"strings"
	"syscall"

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

// formatErrorNoPath formats an OS error without including any file path.
// Lua 5.4 os.rename returns just the error description (e.g., "No such file or directory").
func formatErrorNoPath(err error) (string, int) {
	var errno int
	var errMsg string

	if pathErr, ok := err.(*os.PathError); ok {
		if sysErr, ok := pathErr.Err.(syscall.Errno); ok {
			errno = int(sysErr)
		} else {
			errno = extractErrnoFromError(err)
		}
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
		errMsg = capitalizeError(err.Error())
	}

	return errMsg, errno
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
	caps := provider.Capabilities(v.Context())

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

	// setlocale routes through the OS provider
	osTable.SetString("setlocale", vm.NewNativeFunc(makeOsSetlocale(provider)))

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
		if !cmd.IsString() && !cmd.IsNumber() {
			callerArgError(v, 1, "os.execute", fmt.Sprintf("string expected, got %s", cmd.Type()))
		}
		ok, exitType, exitCode := provider.Execute(v.Context(), valueToString(cmd))
		if err := v.CheckInterrupt(); err != nil {
			panic(err)
		}
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
		v.Set(0, vm.NewFloat(provider.Clock(v.Context())))
		return 1
	}
}

// makeOsTime creates the os.time([table]) function.
func makeOsTime(vmRef *vm.VM, provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		arg := v.Get(1)
		if arg.IsNil() {
			ctx := v.Context()
			ts, _, err := provider.Time(ctx, nil)
			if err != nil {
				panic(err.Error())
			}
			v.Set(0, vm.NewInt(ts))
			return 1
		}

		if !arg.IsTable() {
			callerArgError(v, 1, "os.time", fmt.Sprintf("table expected, got %s", arg.Type()))
		}

		dateTable := &vm.LuaTimeInput{}

		// Required fields
		for _, key := range []string{"year", "month", "day"} {
			val, err := v.IndexValue(arg, vm.NewString(key))
			if err != nil {
				panic(err)
			}
			if val.IsNil() {
				panic(fmt.Sprintf("field '%s' missing in date table", key))
			}
			i, ok := val.ToInt()
			if !ok {
				panic(fmt.Sprintf("field '%s' is not an integer", key))
			}
			switch key {
			case "year":
				dateTable.Year = int(i)
			case "month":
				dateTable.Month = int(i)
			case "day":
				dateTable.Day = int(i)
			}
		}

		// Lua 5.4: validate year fits in C's int (tm_year = year - 1900)
		tmYear := int64(dateTable.Year) - 1900
		if tmYear < math.MinInt32 || tmYear > math.MaxInt32 {
			panic("field 'year' is out-of-bound")
		}

		// Optional fields with Lua 5.4 defaults: hour=12, min=0, sec=0
		defaults := map[string]int{"hour": 12, "min": 0, "sec": 0}
		for _, key := range []string{"hour", "min", "sec"} {
			val, err := v.IndexValue(arg, vm.NewString(key))
			if err != nil {
				panic(err)
			}
			if !val.IsNil() {
				i, ok := val.ToInt()
				if !ok {
					panic(fmt.Sprintf("field '%s' is not an integer", key))
				}
				switch key {
				case "hour":
					dateTable.Hour = int(i)
				case "min":
					dateTable.Min = int(i)
				case "sec":
					dateTable.Sec = int(i)
				}
			} else {
				switch key {
				case "hour":
					dateTable.Hour = defaults[key]
				case "min":
					dateTable.Min = defaults[key]
				case "sec":
					dateTable.Sec = defaults[key]
				}
			}
		}

		isdstVal, err := v.IndexValue(arg, vm.NewString("isdst"))
		if err != nil {
			panic(err)
		}
		if !isdstVal.IsNil() {
			dateTable.HasIsDST = true
			dateTable.IsDST = isdstVal.ToBool()
		}

		ts, norm, err := provider.Time(v.Context(), dateTable)
		if err != nil {
			panic(err.Error())
		}

		if norm != nil {
			setOSDateField(v, arg, "year", vm.NewInt(int64(norm.Year)))
			setOSDateField(v, arg, "month", vm.NewInt(int64(norm.Month)))
			setOSDateField(v, arg, "day", vm.NewInt(int64(norm.Day)))
			setOSDateField(v, arg, "hour", vm.NewInt(int64(norm.Hour)))
			setOSDateField(v, arg, "min", vm.NewInt(int64(norm.Min)))
			setOSDateField(v, arg, "sec", vm.NewInt(int64(norm.Sec)))
			setOSDateField(v, arg, "yday", vm.NewInt(int64(norm.Yday)))
			setOSDateField(v, arg, "wday", vm.NewInt(int64(norm.Wday)))
			if norm.HasDST {
				setOSDateField(v, arg, "isdst", vm.NewBool(norm.IsDST))
			}
		}

		v.Set(0, vm.NewInt(ts))
		return 1
	}
}

// osDifftime implements os.difftime(t2, t1).
func osDifftime(v *vm.VM) int {
	t2 := getInt(v, 1, "os.difftime")
	t1 := getInt(v, 2, "os.difftime")
	v.Set(0, vm.NewFloat(float64(t2-t1)))
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
				panic(fmt.Sprintf("bad argument #2 to 'os.date' (number expected, got %s)", v.Get(2).Type()))
			}
			timestamp = ts
		} else {
			ts, _, _ := provider.Time(v.Context(), nil)
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
			dt := provider.DateTable(v.Context(), timestamp, utc)
			// Match Lua 5.4: reject timestamps that C's gmtime/localtime can't represent.
			if dt.Year > math.MaxInt32+1900 || dt.Year < math.MinInt32+1900 {
				panic("date result cannot be represented in this installation")
			}
			t := vm.NewEmptyTable()
			t.SetString("year", vm.NewInt(int64(dt.Year)))
			t.SetString("month", vm.NewInt(int64(dt.Month)))
			t.SetString("day", vm.NewInt(int64(dt.Day)))
			t.SetString("hour", vm.NewInt(int64(dt.Hour)))
			t.SetString("min", vm.NewInt(int64(dt.Min)))
			t.SetString("sec", vm.NewInt(int64(dt.Sec)))
			t.SetString("wday", vm.NewInt(int64(dt.Wday)))
			t.SetString("yday", vm.NewInt(int64(dt.Yday)))
			t.SetString("isdst", vm.NewBool(dt.IsDST))
			v.Set(0, vm.NewTable(t))
			return 1
		}

		result, err := provider.Date(v.Context(), format, timestamp)
		if err != nil {
			errMsg := err.Error()
			if strings.HasPrefix(errMsg, "date result") {
				// Range error: Lua 5.4 uses luaL_error (no "bad argument" prefix)
				panic(errMsg)
			}
			// Format specifier error: Lua 5.4 uses luaL_argerror
			panic(fmt.Sprintf("bad argument #1 to 'os.date' (%s)", errMsg))
		}
		v.Set(0, vm.NewString(result))
		return 1
	}
}

func setOSDateField(v *vm.VM, tableVal vm.Value, key string, value vm.Value) {
	if err := v.SetIndexValue(tableVal, vm.NewString(key), value); err != nil {
		panic(err)
	}
}

// makeOsTmpname creates the os.tmpname() function.
func makeOsTmpname(ioProvider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name, err := ioProvider.TmpName(v.Context())
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
		if v.ArgCount() < 1 {
			callerArgError(v, 1, "os.remove", "string expected, got no value")
		}
		if name.IsNil() {
			callerArgError(v, 1, "os.remove", "string expected, got nil")
		}
		if !name.IsString() && !name.IsNumber() {
			callerArgError(v, 1, "os.remove", fmt.Sprintf("string expected, got %s", name.Type()))
		}
		nameStr := valueToString(name)
		err := ioProvider.Remove(v.Context(), nameStr)
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

// makeOsSetlocale creates os.setlocale([locale [, category]]).
func makeOsSetlocale(provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		locale := "\x00query" // sentinel: nil arg means query current locale
		if v.ArgCount() >= 1 && !v.Get(1).IsNil() {
			arg1 := v.Get(1)
			if !arg1.IsString() && !arg1.IsNumber() {
				callerArgError(v, 1, "os.setlocale", fmt.Sprintf("string expected, got %s", arg1.Type()))
			}
			locale = valueToString(arg1)
		}

		category := "all"
		if v.ArgCount() >= 2 && !v.Get(2).IsNil() {
			arg2 := v.Get(2)
			if !arg2.IsString() && !arg2.IsNumber() {
				callerArgError(v, 2, "os.setlocale", fmt.Sprintf("string expected, got %s", arg2.Type()))
			}
			category = valueToString(arg2)
			switch category {
			case "all", "collate", "ctype", "monetary", "numeric", "time":
				// valid
			default:
				callerArgError(v, 2, "os.setlocale", fmt.Sprintf("invalid option '%s'", category))
			}
		}

		if result, ok := provider.SetLocale(v.Context(), locale, category); ok {
			v.Set(0, vm.NewString(result))
			return 1
		}

		v.Set(0, vm.Nil)
		return 1
	}
}

// makeOsRename creates the os.rename(oldname, newname) function.
func makeOsRename(ioProvider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		arg1 := v.Get(1)
		if v.ArgCount() < 1 {
			callerArgError(v, 1, "os.rename", "string expected, got no value")
		}
		if arg1.IsNil() {
			callerArgError(v, 1, "os.rename", "string expected, got nil")
		}
		if !arg1.IsString() && !arg1.IsNumber() {
			callerArgError(v, 1, "os.rename", fmt.Sprintf("string expected, got %s", arg1.Type()))
		}
		arg2 := v.Get(2)
		if v.ArgCount() < 2 {
			callerArgError(v, 2, "os.rename", "string expected, got no value")
		}
		if arg2.IsNil() {
			callerArgError(v, 2, "os.rename", "string expected, got nil")
		}
		if !arg2.IsString() && !arg2.IsNumber() {
			callerArgError(v, 2, "os.rename", fmt.Sprintf("string expected, got %s", arg2.Type()))
		}
		oldname := valueToString(arg1)
		newname := valueToString(arg2)
		err := ioProvider.Rename(v.Context(), oldname, newname)
		if err != nil {
			errMsg, errno := formatErrorNoPath(err)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(errMsg))
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
				i := getInt(v, 1, "os.exit")
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

		handler.Exit(v.Context(), code, closeFlag)
		return 0 // unreachable
	}
}

// makeOsGetenv creates the os.getenv(varname) function.
func makeOsGetenv(provider vm.LuaOsProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1)
		if v.ArgCount() < 1 {
			callerArgError(v, 1, "os.getenv", "string expected, got no value")
		}
		if name.IsNil() {
			callerArgError(v, 1, "os.getenv", "string expected, got nil")
		}
		if !name.IsString() && !name.IsNumber() {
			callerArgError(v, 1, "os.getenv", fmt.Sprintf("string expected, got %s", name.Type()))
		}

		val, ok := provider.Getenv(v.Context(), valueToString(name))
		if !ok {
			v.Set(0, vm.Nil)
			return 1
		}
		v.Set(0, vm.NewString(val))
		return 1
	}
}
