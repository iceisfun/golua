package stdlib

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/iceisfun/golua/vm"
)

// print(...)
func luaPrint(v *vm.VM) int {
	n := v.ArgCount()
	var parts []string
	for i := 1; i <= n; i++ {
		parts = append(parts, tolstring(v, v.Get(i)))
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
	if v.ArgCount() < 1 {
		panic("bad argument #1 to 'assert' (value expected)")
	}
	val := v.Get(1)
	if !val.ToBool() {
		if v.ArgCount() < 2 {
			msg := "assertion failed!"
			if loc := v.GetSourceLocation(1); loc != "" {
				msg = loc + ": " + msg
			}
			panic(&vm.LuaError{Value: vm.NewString(msg)})
		}
		msg := v.Get(2)
		if msg.IsString() {
			s := msg.AsString()
			if loc := v.GetSourceLocation(1); loc != "" {
				s = loc + ": " + s
			}
			panic(&vm.LuaError{Value: vm.NewString(s)})
		}
		panic(&vm.LuaError{Value: msg})
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
	if v.ArgCount() < 1 {
		panic("bad argument #1 to 'type' (value expected)")
	}
	val := v.Get(1)
	v.Set(0, vm.NewString(val.Type()))
	return 1
}

// tostring(v)
func luaToString(v *vm.VM) int {
	if v.ArgCount() < 1 {
		panic("bad argument #1 to 'tostring' (value expected)")
	}
	val := v.Get(1)
	// Check for __tostring metamethod on tables
	if val.IsTable() {
		if mt := val.AsTable().Metatable(); mt != nil {
			if ts := mt.Get(vm.NewString("__tostring")); !ts.IsNil() {
				results, err := v.ProtectedCall(ts, []vm.Value{val})
				if err != nil {
					panic(err)
				}
				if len(results) == 0 {
					panic("'__tostring' must return a string")
				}
				ret := results[0]
				if ret.IsString() || ret.IsNumber() {
					v.Set(0, vm.NewString(valueToString(ret)))
					return 1
				}
				panic("'__tostring' must return a string")
			}
		}
	}
	v.Set(0, vm.NewString(valueToString(val)))
	return 1
}

// tonumber(e [, base])
func luaToNumber(v *vm.VM) int {
	if v.ArgCount() < 1 {
		panic("bad argument #1 to 'tonumber' (value expected)")
	}
	val := v.Get(1)
	base := v.Get(2)

	if !base.IsNil() {
		bi := getInt(v, 2, "tonumber")
		if bi < 2 || bi > 36 {
			panic("bad argument #2 to 'tonumber' (base out of range)")
		}
		// Lua semantics: with explicit base, first arg must be a string.
		if !val.IsString() {
			panic(fmt.Sprintf("bad argument #1 to 'tonumber' (string expected, got %s)", val.Type()))
		}
		s := strings.TrimSpace(val.AsString())
		if i, err := strconv.ParseInt(s, int(bi), 64); err == nil {
			v.Set(0, vm.NewInt(i))
			return 1
		}
		// Try unsigned — Lua 5.4 wraps values > INT64_MAX to signed
		if u, err := strconv.ParseUint(s, int(bi), 64); err == nil {
			v.Set(0, vm.NewInt(int64(u)))
			return 1
		}

		v.Set(0, vm.Nil)
		return 1
	}

	if val.IsNumber() {
		v.Set(0, val)
		return 1
	}

	if val.IsString() {
		trimmed := strings.TrimSpace(val.AsString())
		if trimmed == "" {
			v.Set(0, vm.Nil)
			return 1
		}
		// Lua does not accept underscore separators in tonumber input.
		if strings.ContainsRune(trimmed, '_') {
			v.Set(0, vm.Nil)
			return 1
		}
		// Lua rejects textual inf/nan tokens in tonumber input.
		lower := strings.ToLower(trimmed)
		switch lower {
		case "inf", "+inf", "-inf", "nan", "+nan", "-nan":
			v.Set(0, vm.Nil)
			return 1
		}

		// Signed/unsigned hex integer forms (e.g. -0x10, +0XFF).
		sign := int64(1)
		body := trimmed
		if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
			if body[0] == '-' {
				sign = -1
			}
			body = body[1:]
		}
		// Lua 5.4: base-10 tonumber accepts 0x/0X hex prefix
		if len(body) >= 2 && body[0] == '0' && (body[1] == 'x' || body[1] == 'X') {
			hex := body[2:]
			if hex != "" && !strings.ContainsAny(hex, ".pP") {
				if u, err := strconv.ParseUint(hex, 16, 64); err == nil {
					v.Set(0, vm.NewInt(sign*int64(u)))
					return 1
				}
			}
			// Hex float forms (e.g. 0x1p4, 0x.1, 0xA.8) — try ParseHexFloat
			if f, ok := vm.ParseHexFloat(trimmed); ok {
				v.Set(0, vm.NewFloat(f))
				return 1
			}
		}
		// If the string looks like a pure decimal integer, try integer parsing first
		// to preserve int64 precision.
		isDecIntLike := true
		for i, c := range trimmed {
			if i == 0 && (c == '+' || c == '-') {
				continue
			}
			if c < '0' || c > '9' {
				isDecIntLike = false
				break
			}
		}
		if isDecIntLike {
			if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
				v.Set(0, vm.NewInt(i))
				return 1
			}
		}
		// Try float
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil || errors.Is(err, strconv.ErrRange) {
			v.Set(0, vm.NewFloat(f))
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
	argc := v.ArgCount()
	if argc < 1 {
		panic("bad argument #1 to 'pcall' (value expected)")
	}
	fn := v.Get(1)

	// Collect additional arguments
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
		panic(fmt.Sprintf("bad argument #2 to 'xpcall' (function expected, got %s)", msgh.Type()))
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
			v.Set(1, vm.NewString("error in error handling"))
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
		panic(fmt.Sprintf("bad argument #1 to 'collectgarbage' (invalid option '%s')", opt))
	}
}

// warn(msg1 [, msg2, ...])
func luaWarn(v *vm.VM) int {
	argc := v.ArgCount()
	if argc < 1 {
		panic("bad argument #1 to 'warn' (string expected, got no value)")
	}
	first := v.Get(1)
	if !first.IsString() {
		panic(fmt.Sprintf("bad argument #1 to 'warn' (string expected, got %s)", first.Type()))
	}
	s := first.AsString()
	if len(s) > 0 && s[0] == '@' {
		switch s {
		case "@off":
			v.SetWarnEnabled(false)
		case "@on":
			v.SetWarnEnabled(true)
		}
		return 0
	}
	if v.WarnEnabled() {
		var buf strings.Builder
		buf.WriteString("Lua warning: ")
		buf.WriteString(s)
		for i := 2; i <= argc; i++ {
			arg := v.Get(i)
			if !arg.IsString() {
				panic(fmt.Sprintf("bad argument #%d to 'warn' (string expected, got %s)", i, arg.Type()))
			}
			buf.WriteString(arg.AsString())
		}
		v.Warn(buf.String())
	}
	return 0
}

// pairs(t)
func luaPairs(v *vm.VM) int {
	arg := v.Get(1)

	// Check for __pairs metamethod (only on tables)
	if arg.IsTable() {
		tbl := arg.AsTable()
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
	}

	// Return next, t, nil — next will validate the table arg
	v.Set(0, vm.NewNativeFunc(luaNext))
	v.Set(1, arg)
	v.Set(2, vm.Nil)
	return 3
}

// ipairs(t)
func luaIpairs(v *vm.VM) int {
	arg := v.Get(1)

	// ipairs iterator — uses metamethod-aware access so __index is honored
	iter := vm.NewNativeFunc(func(v *vm.VM) int {
		t := v.Get(1).AsTable()
		if t == nil {
			panic(fmt.Sprintf("bad argument #1 to 'ipairs' (table expected, got %s)", v.Get(1).Type()))
		}
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
	v.Set(1, arg)
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
	nextK, nextV, err := tbl.Next(key)
	if err != nil {
		panic(err.Error())
	}
	if nextK.IsNil() {
		v.Set(0, vm.Nil)
		return 1
	}
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
	if err := tbl.Set(key, val); err != nil {
		panic(err.Error())
	}
	v.Set(0, vm.NewTable(tbl))
	return 1
}

// rawequal(v1, v2)
func luaRawequal(v *vm.VM) int {
	if v.ArgCount() < 1 {
		panic("bad argument #1 to 'rawequal' (value expected)")
	}
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
	if v.ArgCount() < 1 {
		panic("bad argument #1 to 'getmetatable' (value expected)")
	}
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

// tolstring resolves __tostring metamethods, matching luaL_tolstring.
func tolstring(v *vm.VM, val vm.Value) string {
	if val.IsTable() {
		if mt := val.AsTable().Metatable(); mt != nil {
			if ts := mt.Get(vm.NewString("__tostring")); !ts.IsNil() {
				results, err := v.ProtectedCall(ts, []vm.Value{val})
				if err != nil {
					panic(err)
				}
				if len(results) == 0 {
					panic("'__tostring' must return a string")
				}
				ret := results[0]
				if ret.IsString() || ret.IsNumber() {
					return valueToString(ret)
				}
				panic("'__tostring' must return a string")
			}
		}
	}
	return valueToString(val)
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
		if val.IsInt() {
			return val.String()
		}
		f := val.AsFloat()
		if math.IsInf(f, 1) {
			return "inf"
		}
		if math.IsInf(f, -1) {
			return "-inf"
		}
		if math.IsNaN(f) {
			if math.Signbit(f) {
				return "-nan"
			}
			return "nan"
		}
		return val.String()
	case val.IsString():
		return val.AsString()
	case val.IsTable():
		tbl := val.AsTable()
		if t, ok := tbl.(*vm.Table); ok && t.IsThread() {
			return fmt.Sprintf("thread: %p", tbl)
		}
		return fmt.Sprintf("table: %p", tbl)
	case val.IsFunction():
		return fmt.Sprintf("function: %p", val.AsClosure())
	case val.IsNativeFunc():
		return "function: (native)"
	default:
		return "?"
	}
}
