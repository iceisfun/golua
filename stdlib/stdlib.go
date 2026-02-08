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
}

// print(...)
func luaPrint(v *vm.VM) int {
	n := v.ArgCount()
	var parts []string
	for i := 1; i <= n; i++ {
		parts = append(parts, valueToString(v.Get(i)))
	}
	fmt.Println(strings.Join(parts, "\t"))
	return 0
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
			// Try float first
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				v.Set(0, vm.NewFloat(f))
				return 1
			}
		}

		// Try integer with base
		if i, err := strconv.ParseInt(s, b, 64); err == nil {
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
	panic(valueToString(msg))
}

// pcall(f [, arg1, ...])
func luaPcall(v *vm.VM) int {
	fn := v.Get(1)
	if !fn.IsFunction() && !fn.IsNativeFunc() {
		panic("bad argument #1 to 'pcall' (function expected)")
	}

	// Collect additional arguments
	argc := v.ArgCount()
	args := make([]vm.Value, argc-1)
	for i := 2; i <= argc; i++ {
		args[i-2] = v.Get(i)
	}

	// Call the function with error protection
	results, err := v.ProtectedCall(fn, args)
	if err != nil {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}

	// Success: return true followed by all results
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
		// Trigger GC (no-op, Go handles this)
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
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'pairs' (table expected)")
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

	// ipairs iterator
	iter := vm.NewNativeFunc(func(v *vm.VM) int {
		t := v.Get(1).AsTable()
		i := v.Get(2)
		idx, _ := i.ToInt()
		idx++
		val := t.GetInt(int(idx))
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
			protected := mt.GetString("__metatable")
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
	v.Set(0, vm.Nil)
	return 1
}

// setmetatable(table, metatable)
func luaSetmetatable(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'setmetatable' (table expected)")
	}
	mt := v.Get(2)
	if mt.IsNil() {
		tbl.SetMetatable(nil)
	} else if mt.IsTable() {
		tbl.SetMetatable(mt.AsTable())
	} else {
		panic("bad argument #2 to 'setmetatable' (nil or table expected)")
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
