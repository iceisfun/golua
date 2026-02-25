package stdlib

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iceisfun/golua/vm"
)

// Open registers all standard library functions in the VM.
func Open(v *vm.VM) {
	// Basic functions
	v.SetGlobal("print", vm.NewNativeFunc(luaPrint))
	v.SetGlobal("assert", vm.NewNativeFunc(luaAssert))
	v.SetGlobal("type", vm.NewNativeFunc(luaType))
	v.SetGlobal("tostring", vm.NewNativeFunc(luaToString))
	v.SetGlobal("tonumber", vm.NewNativeFunc(luaToNumber))
	v.SetGlobal("error", vm.NewNativeFunc(luaError))
	v.SetGlobal("pcall", vm.NewNativeFunc(luaPcall))
	v.SetGlobal("xpcall", vm.NewNativeFunc(luaXpcall))
	v.SetGlobal("pairs", vm.NewNativeFunc(luaPairs))
	v.SetGlobal("ipairs", vm.NewNativeFunc(luaIpairs))
	v.SetGlobal("next", vm.NewNativeFunc(luaNext))
	v.SetGlobal("select", vm.NewNativeFunc(luaSelect))
	v.SetGlobal("rawget", vm.NewNativeFunc(luaRawget))
	v.SetGlobal("rawset", vm.NewNativeFunc(luaRawset))
	v.SetGlobal("rawequal", vm.NewNativeFunc(luaRawequal))
	v.SetGlobal("rawlen", vm.NewNativeFunc(luaRawlen))
	v.SetGlobal("getmetatable", vm.NewNativeFunc(luaGetmetatable))
	v.SetGlobal("setmetatable", vm.NewNativeFunc(luaSetmetatable))
	v.SetGlobal("collectgarbage", vm.NewNativeFunc(luaCollectgarbage))

	// Output inspection (for testing)
	v.SetGlobal("_lastoutput", vm.NewNativeFunc(luaLastOutput))
	v.SetGlobal("_outputlines", vm.NewNativeFunc(luaOutputLines))

	// _G points to the globals table
	v.SetGlobal("_G", vm.NewTable(v.Globals()))

	// _VERSION
	v.SetGlobal("_VERSION", vm.NewString("Lua 5.5"))

	// String library
	openString(v)

	// Math library
	openMath(v)

	// Table library
	openTable(v)

	// Coroutine library
	openCoroutine(v)

	// Load functions (load, loadfile, dofile)
	openLoad(v)

	// IO library (only if IoProvider is set)
	openIo(v)

	// OS library (only if OsProvider is set)
	openOs(v)

	// Debug library (only if DebugProvider is set)
	openDebug(v)

	// Channel library (only if ChanProvider is set)
	openChan(v)

	// Bit32 library (Lua 5.2 compat)
	openBit32(v)

	// UTF-8 library (strict mode only)
	openUtf8(v)

	// Glob library (Go-style pattern matching)
	openGlob(v)

	// Time library (only if TimeProvider is set)
	openTime(v)
}

// print(...)
func luaPrint(v *vm.VM) int {
	n := v.ArgCount()
	var parts []string
	for i := 1; i <= n; i++ {
		parts = append(parts, valueToString(v.Get(i)))
	}
	v.Print(strings.Join(parts, "\t"))
	return 0
}

// _lastoutput() returns the most recent captured output line (requires WithCaptureOutput).
func luaLastOutput(v *vm.VM) int {
	v.Set(0, vm.NewString(v.LastOutput()))
	return 1
}

// _outputlines() returns a table of all captured output lines (requires WithCaptureOutput).
func luaOutputLines(v *vm.VM) int {
	lines := v.OutputLines()
	t := vm.NewEmptyTable()
	for i, line := range lines {
		t.SetInt(i+1, vm.NewString(line))
	}
	v.Set(0, vm.NewTable(t))
	return 1
}

// assert(v [, message])
func luaAssert(v *vm.VM) int {
	val := v.Get(1)
	if !val.ToBool() {
		msg := v.Get(2)
		if msg.IsNil() {
			panic("assertion failed!")
		}
		panic(valueToString(msg))
	}
	// Return all arguments
	n := v.ArgCount()
	for i := 1; i <= n; i++ {
		v.Set(i-1, v.Get(i))
	}
	return n
}

// type(v)
func luaType(v *vm.VM) int {
	val := v.Get(1)
	v.Set(0, vm.NewString(val.Type()))
	return 1
}

// tostring(v)
func luaToString(v *vm.VM) int {
	val := v.Get(1)
	// Check for __tostring metamethod on tables
	if val.IsTable() {
		if mt := val.AsTable().Metatable(); mt != nil {
			if ts := mt.Get(vm.NewString("__tostring")); !ts.IsNil() {
				results, err := v.ProtectedCall(ts, []vm.Value{val})
				if err != nil {
					panic(err)
				}
				if len(results) > 0 {
					v.Set(0, results[0])
				} else {
					v.Set(0, vm.NewString("nil"))
				}
				return 1
			}
		}
	}
	v.Set(0, vm.NewString(valueToString(val)))
	return 1
}

// tonumber(e [, base])
func luaToNumber(v *vm.VM) int {
	val := v.Get(1)
	base := v.Get(2)

	if val.IsNumber() {
		v.Set(0, val)
		return 1
	}

	if val.IsString() {
		s := val.AsString()
		b := 10
		if !base.IsNil() {
			if bi, ok := base.ToInt(); ok {
				b = int(bi)
			}
		}

		if b == 10 {
			trimmed := strings.TrimSpace(s)
			// Lua 5.4: base-10 tonumber accepts 0x/0X hex prefix
			if len(trimmed) >= 2 && trimmed[0] == '0' && (trimmed[1] == 'x' || trimmed[1] == 'X') {
				if i, err := strconv.ParseInt(trimmed[2:], 16, 64); err == nil {
					v.Set(0, vm.NewInt(i))
					return 1
				}
			}
			// If the string looks like a pure integer (no '.', 'e', 'E', 'n', 'N'),
			// try integer parsing first to preserve int64 precision
			isIntLike := true
			for _, c := range trimmed {
				if c == '.' || c == 'e' || c == 'E' || c == 'n' || c == 'N' {
					isIntLike = false
					break
				}
			}
			if isIntLike {
				if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
					v.Set(0, vm.NewInt(i))
					return 1
				}
			}
			// Try float
			if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
				v.Set(0, vm.NewFloat(f))
				return 1
			}
		}

		// Try integer with base
		if i, err := strconv.ParseInt(strings.TrimSpace(s), b, 64); err == nil {
			v.Set(0, vm.NewInt(i))
			return 1
		}
	}

	v.Set(0, vm.Nil)
	return 1
}

// error(message [, level])
func luaError(v *vm.VM) int {
	msg := v.Get(1)
	level := int64(1) // default
	if !v.Get(2).IsNil() {
		level = getInt(v, 2, "error")
	}

	// For string messages with level > 0, prepend source location
	if level > 0 && msg.IsString() {
		if loc := v.GetSourceLocation(int(level)); loc != "" {
			msg = vm.NewString(loc + ": " + msg.AsString())
		}
	}
	panic(&vm.LuaError{Value: msg})
}

// pcall(f [, arg1, ...])
func luaPcall(v *vm.VM) int {
	fn := v.Get(1)

	// Collect additional arguments
	argc := v.ArgCount()
	args := make([]vm.Value, argc-1)
	for i := 2; i <= argc; i++ {
		args[i-2] = v.Get(i)
	}

	// Call the function with error protection
	// ProtectedCall handles __call metamethods for tables
	results, err := v.ProtectedCall(fn, args)
	if err != nil {
		v.Set(0, vm.False)
		// Preserve the original Lua error value if available
		if le, ok := err.(*vm.LuaError); ok {
			v.Set(1, le.Value)
		} else {
			v.Set(1, vm.NewString(err.Error()))
		}
		return 2
	}

	// Success: return true followed by all results
	v.EnsureStack(v.Base() + 1 + len(results))
	v.Set(0, vm.True)
	for i, r := range results {
		v.Set(i+1, r)
	}
	return 1 + len(results)
}

// xpcall(f, msgh [, arg1, ...])
func luaXpcall(v *vm.VM) int {
	fn := v.Get(1)
	msgh := v.Get(2)
	if !msgh.IsFunction() && !msgh.IsNativeFunc() {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("attempt to call a "+msgh.Type()+" value"))
		return 2
	}

	// Collect extra arguments (after fn and msgh)
	argc := v.ArgCount()
	args := make([]vm.Value, argc-2)
	for i := 3; i <= argc; i++ {
		args[i-3] = v.Get(i)
	}

	results, err := v.ProtectedCall(fn, args)
	if err != nil {
		// Pass the original Lua error value to the message handler
		var errVal vm.Value
		if le, ok := err.(*vm.LuaError); ok {
			errVal = le.Value
		} else {
			errVal = vm.NewString(err.Error())
		}
		handlerResults, handlerErr := v.ProtectedCall(msgh, []vm.Value{errVal})
		v.Set(0, vm.False)
		if handlerErr != nil {
			v.Set(1, vm.NewString(handlerErr.Error()))
		} else if len(handlerResults) > 0 {
			v.Set(1, handlerResults[0])
		} else {
			v.Set(1, vm.Nil)
		}
		return 2
	}

	// Success: return true followed by all results
	v.EnsureStack(v.Base() + 1 + len(results))
	v.Set(0, vm.True)
	for i, r := range results {
		v.Set(i+1, r)
	}
	return 1 + len(results)
}

// collectgarbage([opt [, arg]])
func luaCollectgarbage(v *vm.VM) int {
	// Go handles garbage collection automatically
	// This is a no-op stub that returns 0 for compatibility
	opt := ""
	if !v.Get(1).IsNil() {
		opt = v.Get(1).AsString()
	}

	switch opt {
	case "collect", "":
		v.ProcessGcFinalizers()
		v.Set(0, vm.NewInt(0))
		return 1
	case "count":
		// Return approximate memory in use (in KB)
		v.Set(0, vm.NewInt(0))
		return 1
	case "stop", "restart":
		// No-op
		return 0
	case "isrunning":
		v.Set(0, vm.True)
		return 1
	default:
		v.Set(0, vm.NewInt(0))
		return 1
	}
}

// pairs(t)
func luaPairs(v *vm.VM) int {
	arg := v.Get(1)
	tbl := arg.AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'pairs' (table expected)")
	}

	// Check for __pairs metamethod
	if mt := tbl.Metatable(); mt != nil {
		if mp := mt.Get(vm.NewString("__pairs")); !mp.IsNil() {
			results, err := v.ProtectedCall(mp, []vm.Value{arg})
			if err != nil {
				panic(err)
			}
			for i, r := range results {
				v.Set(i, r)
			}
			return len(results)
		}
	}

	// Return next, t, nil
	v.Set(0, vm.NewNativeFunc(luaNext))
	v.Set(1, vm.NewTable(tbl))
	v.Set(2, vm.Nil)
	return 3
}

// ipairs(t)
func luaIpairs(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'ipairs' (table expected)")
	}

	// ipairs iterator — uses metamethod-aware access so __index is honored
	iter := vm.NewNativeFunc(func(v *vm.VM) int {
		t := v.Get(1).AsTable()
		i := v.Get(2)
		idx, _ := i.ToInt()
		idx++
		val, err := v.TableGetInt(t, int(idx))
		if err != nil {
			panic(err)
		}
		if val.IsNil() {
			v.Set(0, vm.Nil)
			return 1
		}
		v.Set(0, vm.NewInt(idx))
		v.Set(1, val)
		return 2
	})

	v.Set(0, iter)
	v.Set(1, vm.NewTable(tbl))
	v.Set(2, vm.NewInt(0))
	return 3
}

// next(table [, index])
func luaNext(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'next' (table expected)")
	}
	key := v.Get(2)
	nextK, nextV := tbl.Next(key)
	v.Set(0, nextK)
	v.Set(1, nextV)
	return 2
}

// select(index, ...)
func luaSelect(v *vm.VM) int {
	idx := v.Get(1)
	n := v.ArgCount()

	if idx.IsString() && idx.AsString() == "#" {
		v.Set(0, vm.NewInt(int64(n-1)))
		return 1
	}

	i, ok := idx.ToInt()
	if !ok {
		panic("bad argument #1 to 'select' (number expected)")
	}

	if i < 0 {
		i = int64(n) + i
	}

	if i < 1 {
		panic("bad argument #1 to 'select' (index out of range)")
	}

	// Return values from index i onwards
	if c := n - 1 - int(i) + 1; c > 0 {
		v.EnsureStack(v.Base() + c)
	}
	count := 0
	for j := int(i); j <= n-1; j++ {
		v.Set(count, v.Get(j+1))
		count++
	}
	return count
}

// rawget(table, index)
func luaRawget(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'rawget' (table expected)")
	}
	key := v.Get(2)
	v.Set(0, tbl.Get(key))
	return 1
}

// rawset(table, index, value)
func luaRawset(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'rawset' (table expected)")
	}
	key := v.Get(2)
	val := v.Get(3)
	tbl.Set(key, val)
	v.Set(0, vm.NewTable(tbl))
	return 1
}

// rawequal(v1, v2)
func luaRawequal(v *vm.VM) int {
	v1 := v.Get(1)
	v2 := v.Get(2)
	v.Set(0, vm.NewBool(v1.Equal(v2)))
	return 1
}

// rawlen(v)
func luaRawlen(v *vm.VM) int {
	val := v.Get(1)
	if val.IsString() {
		v.Set(0, vm.NewInt(int64(len(val.AsString()))))
		return 1
	}
	if val.IsTable() {
		v.Set(0, vm.NewInt(int64(val.AsTable().Len())))
		return 1
	}
	panic("bad argument #1 to 'rawlen' (table or string expected)")
}

// getmetatable(object)
func luaGetmetatable(v *vm.VM) int {
	val := v.Get(1)
	if val.IsTable() {
		mt := val.AsTable().Metatable()
		if mt != nil {
			// Check for __metatable field - if present, return that instead
			protected := mt.Get(vm.NewString("__metatable"))
			if !protected.IsNil() {
				v.Set(0, protected)
				return 1
			}
			v.Set(0, vm.NewTable(mt))
		} else {
			v.Set(0, vm.Nil)
		}
		return 1
	}
	if val.IsString() {
		if sm := v.StringMeta(); sm != nil {
			v.Set(0, vm.NewTable(sm))
		} else {
			v.Set(0, vm.Nil)
		}
		return 1
	}
	v.Set(0, vm.Nil)
	return 1
}

// setmetatable(table, metatable)
func luaSetmetatable(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'setmetatable' (table expected)")
	}
	// Check __metatable protection on existing metatable
	if existingMT := tbl.Metatable(); existingMT != nil {
		if !existingMT.Get(vm.NewString("__metatable")).IsNil() {
			panic("cannot change a protected metatable")
		}
	}
	mt := v.Get(2)
	if mt.IsNil() {
		tbl.SetMetatable(nil)
	} else if mt.IsTable() {
		tbl.SetMetatable(mt.AsTable())
	} else {
		panic("bad argument #2 to 'setmetatable' (nil or table expected)")
	}
	// Register __gc finalizer if applicable
	if concrete, ok := tbl.(*vm.Table); ok {
		vm.RegisterGcFinalizer(concrete)
	}
	v.Set(0, vm.NewTable(tbl))
	return 1
}

func valueToString(val vm.Value) string {
	switch {
	case val.IsNil():
		return "nil"
	case val.IsBool():
		if val.AsBool() {
			return "true"
		}
		return "false"
	case val.IsNumber():
		return val.String()
	case val.IsString():
		return val.AsString()
	case val.IsTable():
		return fmt.Sprintf("table: %p", val.AsTable())
	case val.IsFunction():
		return fmt.Sprintf("function: %p", val.AsClosure())
	case val.IsNativeFunc():
		return "function: (native)"
	default:
		return "?"
	}
}
