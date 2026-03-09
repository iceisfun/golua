package stdlib

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/iceisfun/golua/vm"
)

// gotDesc returns "got TYPE" or "got no value" for arg error messages.
func gotDesc(v *vm.VM, argn int) string {
	if v.ArgCount() < argn {
		return ", got no value"
	}
	return fmt.Sprintf(", got %s", v.Get(argn).Type())
}

// print(...)
func luaPrint(v *vm.VM) int {
	n := v.ArgCount()
	// Snapshot all arguments before processing. tolstring may call
	// __tostring metamethods via ProtectedCall, which can shift the
	// native call's stack frame (especially when invoked through pcall).
	args := make([]vm.Value, n)
	for i := 0; i < n; i++ {
		args[i] = v.Get(i + 1)
	}
	var parts []string
	for _, arg := range args {
		parts = append(parts, tolstring(v, arg))
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
		callerArgError(v, 1, "assert", "value expected")
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
		callerArgError(v, 1, "type", "value expected")
	}
	val := v.Get(1)
	v.Set(0, vm.NewString(val.Type()))
	return 1
}

// tostring(v)
func luaToString(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "tostring", "value expected")
	}
	val := v.Get(1)
	// Check for __tostring metamethod on tables, userdata, and type metatables
	var mt vm.LuaTable
	if val.IsTable() {
		mt = val.AsTable().Metatable()
	} else if ud := val.AsUserdata(); ud != nil {
		mt = ud.Metatable()
	}
	if mt == nil {
		mt = v.GetTypeMeta(val)
	}
	if mt != nil {
		if ts := mt.Get(vm.NewString("__tostring")); !ts.IsNil() {
			exitNonYieldable := v.EnterNonYieldable()
			defer exitNonYieldable()
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
	v.Set(0, vm.NewString(valueToString(val)))
	return 1
}

// tonumber(e [, base])
func luaToNumber(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "tonumber", "value expected")
	}
	val := v.Get(1)
	base := v.Get(2)

	if !base.IsNil() {
		// Lua 5.4 order: (1) arg2 integer type, (2) arg1 string type, (3) base range.
		bi := getInt(v, 2, "tonumber")
		if !val.IsString() {
			callerArgError(v, 1, "tonumber", fmt.Sprintf("string expected, got %s", val.Type()))
		}
		if bi < 2 || bi > 36 {
			callerArgError(v, 2, "tonumber", "base out of range")
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
		// Lua 5.4 wraps overflows modulo 2^64 for explicit base
		if result, ok := parseIntWrapping(s, int(bi)); ok {
			v.Set(0, vm.NewInt(result))
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
		bare := lower
		if len(bare) > 0 && (bare[0] == '+' || bare[0] == '-') {
			bare = bare[1:]
		}
		if strings.HasPrefix(bare, "inf") || strings.HasPrefix(bare, "nan") {
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
				// Overflow: parse digit-by-digit with modular wrapping (Lua 5.4)
				var result uint64
				valid := true
				for _, c := range hex {
					var d uint64
					switch {
					case c >= '0' && c <= '9':
						d = uint64(c - '0')
					case c >= 'a' && c <= 'f':
						d = uint64(c-'a') + 10
					case c >= 'A' && c <= 'F':
						d = uint64(c-'A') + 10
					default:
						valid = false
						break
					}
					if !valid {
						break
					}
					result = result*16 + d
				}
				if valid {
					v.Set(0, vm.NewInt(sign*int64(result)))
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
		callerArgError(v, 1, "pcall", "value expected")
	}
	fn := v.Get(1)

	// Collect additional arguments
	args := make([]vm.Value, argc-1)
	for i := 2; i <= argc; i++ {
		args[i-2] = v.Get(i)
	}

	// Save and clear the message handler so that a nested pcall inside
	// xpcall doesn't accidentally trigger the xpcall message handler.
	savedMsgHandler := v.MsgHandler
	savedMsgHandlerUsed := v.MsgHandlerUsed
	savedMsgHandlerResult := v.MsgHandlerResult
	v.MsgHandler = vm.Nil
	v.MsgHandlerUsed = false
	v.MsgHandlerResult = vm.Nil
	exitUserProtected := v.EnterUserProtected()
	defer exitUserProtected()
	if fn.RawEqual(v.GetGlobal("load")) {
		exitDirectLoad := v.EnterDirectProtectedLoad()
		defer exitDirectLoad()
	}

	// Call the function with error protection
	// ProtectedCall handles __call metamethods for tables
	results, err := v.ProtectedCall(fn, args)

	// Restore the outer message handler state
	v.MsgHandler = savedMsgHandler
	v.MsgHandlerUsed = savedMsgHandlerUsed
	v.MsgHandlerResult = savedMsgHandlerResult

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
		got := msgh.Type()
		if v.ArgCount() < 2 {
			got = "no value"
		}
		callerArgError(v, 2, "xpcall", fmt.Sprintf("function expected, got %s", got))
	}

	// Collect extra arguments (after fn and msgh)
	argc := v.ArgCount()
	args := make([]vm.Value, argc-2)
	for i := 3; i <= argc; i++ {
		args[i-3] = v.Get(i)
	}

	// Set message handler so ProtectedCall can call it BEFORE truncating
	// the call stack (allowing debug.traceback to see the full stack).
	v.MsgHandler = msgh
	v.MsgHandlerUsed = false
	v.MsgHandlerResult = vm.Nil
	exitUserProtected := v.EnterUserProtected()
	defer exitUserProtected()
	if fn.RawEqual(v.GetGlobal("load")) {
		exitDirectLoad := v.EnterDirectProtectedLoad()
		defer exitDirectLoad()
	}

	results, err := v.ProtectedCall(fn, args)
	if err != nil {
		v.Set(0, vm.False)
		if v.MsgHandlerUsed {
			// Message handler was already called inside ProtectedCall
			v.Set(1, v.MsgHandlerResult)
		} else {
			// Fallback: call the message handler now (shouldn't normally happen)
			var errVal vm.Value
			if le, ok := err.(*vm.LuaError); ok {
				errVal = le.Value
			} else {
				errVal = vm.NewString(err.Error())
			}
			exitNonYieldable := v.EnterNonYieldable()
			handlerResults, handlerErr := v.ProtectedCall(msgh, []vm.Value{errVal})
			exitNonYieldable()
			if handlerErr != nil {
				v.Set(1, vm.NewString("error in error handling"))
			} else if len(handlerResults) > 0 {
				v.Set(1, handlerResults[0])
			} else {
				v.Set(1, vm.Nil)
			}
		}
		// Clear message handler state
		v.MsgHandler = vm.Nil
		v.MsgHandlerUsed = false
		v.MsgHandlerResult = vm.Nil
		return 2
	}
	// Clear message handler state on success
	v.MsgHandler = vm.Nil
	v.MsgHandlerUsed = false
	v.MsgHandlerResult = vm.Nil

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
	arg1 := v.Get(1)
	if !arg1.IsNil() {
		if !arg1.IsString() {
			callerArgError(v, 1, "collectgarbage", fmt.Sprintf("invalid option '%s'", arg1.AsString()))
		}
		opt = arg1.AsString()
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
		// No-op, return 0 to match Lua 5.4
		v.Set(0, vm.NewInt(0))
		return 1
	case "generational", "incremental":
		// No-op: Go has its own GC. Return previous mode values.
		// Lua 5.4 returns the previous mode name and two mode-specific values.
		v.Set(0, vm.NewInt(0))
		v.Set(1, vm.NewInt(0))
		return 2
	case "step":
		// No-op: trigger a GC step. Return false (step did not finish a cycle).
		v.ProcessGcFinalizers()
		v.Set(0, vm.False)
		return 1
	case "isrunning":
		v.Set(0, vm.True)
		return 1
	default:
		callerArgError(v, 1, "collectgarbage", fmt.Sprintf("invalid option '%s'", opt))
	}
	return 0 // unreachable
}

// warn(msg1 [, msg2, ...])
func luaWarn(v *vm.VM) int {
	argc := v.ArgCount()
	if argc < 1 {
		callerArgError(v, 1, "warn", "string expected, got no value")
	}
	// Lua 5.4: all arguments must be strings or numbers (coerced to string)
	first := v.Get(1)
	if !first.IsString() && !first.IsInt() && !first.IsFloat() {
		callerArgError(v, 1, "warn", fmt.Sprintf("string expected, got %s", first.Type()))
	}
	s := valueToString(first)
	for i := 2; i <= argc; i++ {
		arg := v.Get(i)
		if !arg.IsString() && !arg.IsInt() && !arg.IsFloat() {
			callerArgError(v, i, "warn", fmt.Sprintf("string expected, got %s", arg.Type()))
		}
	}
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
			buf.WriteString(valueToString(v.Get(i)))
		}
		v.Warn(buf.String())
	}
	return 0
}

// pairs(t)
// Lua 5.4: pairs accepts any value (including nil); next() will error later.
// But pairs() with no arguments is an error.
func luaPairs(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "pairs", "value expected")
	}
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
				for i := 0; i < 3; i++ {
					if i < len(results) {
						v.Set(i, results[i])
					} else {
						v.Set(i, vm.Nil)
					}
				}
				return 3
			}
		}
	}

	// Return next, t, nil — next will validate the table arg when called
	v.Set(0, nextFunc)
	v.Set(1, arg)
	v.Set(2, vm.Nil)
	return 3
}

// nextFunc is the shared native function for next(), reused by pairs()
// so that pairs(t) == next holds (identity equality).
var nextFunc = vm.NewNativeFunc(luaNext)

// ipairsIter is the shared iterator function returned by ipairs().
// A single Value is reused so that ipairs{} == ipairs{} holds.
var ipairsIter = vm.NewNativeFunc(func(v *vm.VM) int {
	tval := v.Get(1)
	i := v.Get(2)
	idx, _ := i.ToInt()
	idx++
	// Use IndexValue to support any indexable type and produce natural
	// "attempt to index" errors for non-indexable types (like numbers).
	val, err := v.IndexValue(tval, vm.NewInt(idx))
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

// ipairs(t)
// Lua 5.4: ipairs accepts any value (including nil); the iterator will error later.
// But ipairs() with no arguments is an error.
func luaIpairs(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "ipairs", "value expected")
	}
	arg := v.Get(1)
	v.Set(0, ipairsIter)
	v.Set(1, arg)
	v.Set(2, vm.NewInt(0))
	return 3
}

// next(table [, index])
func luaNext(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		callerArgError(v, 1, "next", "table expected"+gotDesc(v, 1))
	}
	key := v.Get(2)
	nextK, nextV, err := tbl.Next(key)
	if err != nil {
		// Use LuaError to avoid AddCallerLocation adding a file:line prefix.
		// In Lua 5.4, luaG_runerror from C code doesn't add the prefix
		// because the current frame is a C function, not a Lua frame.
		panic(&vm.LuaError{Value: vm.NewString(err.Error())})
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

	if idx.IsString() && len(idx.AsString()) > 0 && idx.AsString()[0] == '#' {
		v.Set(0, vm.NewInt(int64(n-1)))
		return 1
	}

	i, ok := idx.ToInt()
	if !ok {
		if idx.IsNumber() {
			callerArgError(v, 1, "select", "number has no integer representation")
		}
		typeName := idx.Type()
		if n < 1 {
			typeName = "no value"
		}
		callerArgError(v, 1, "select", "number expected, got "+typeName)
	}

	if i < 0 {
		i = int64(n) + i
	}

	if i < 1 {
		callerArgError(v, 1, "select", "index out of range")
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
		callerArgError(v, 1, "rawget", "table expected"+gotDesc(v, 1))
	}
	if v.ArgCount() < 2 {
		callerArgError(v, 2, "rawget", "value expected")
	}
	key := v.Get(2)
	v.Set(0, tbl.Get(key))
	return 1
}

// rawset(table, index, value)
func luaRawset(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		callerArgError(v, 1, "rawset", "table expected"+gotDesc(v, 1))
	}
	if v.ArgCount() < 2 {
		callerArgError(v, 2, "rawset", "value expected")
	}
	if v.ArgCount() < 3 {
		callerArgError(v, 3, "rawset", "value expected")
	}
	key := v.Get(2)
	val := v.Get(3)
	if err := tbl.Set(key, val); err != nil {
		// Use LuaError to avoid AddCallerLocation adding a file:line prefix.
		// These errors originate from native/C-level code, not Lua bytecode.
		panic(&vm.LuaError{Value: vm.NewString(err.Error())})
	}
	v.Set(0, vm.NewTable(tbl))
	return 1
}

// rawequal(v1, v2)
func luaRawequal(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "rawequal", "value expected")
	}
	if v.ArgCount() < 2 {
		callerArgError(v, 2, "rawequal", "value expected")
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
	callerArgError(v, 1, "rawlen", "table or string expected"+gotDesc(v, 1))
	return 0 // unreachable
}

// getmetatable(object)
func luaGetmetatable(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "getmetatable", "value expected")
	}
	val := v.Get(1)
	// Threads use type-level metatable, not per-instance
	if val.IsTable() && !val.AsTable().IsThread() {
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
	// Check userdata per-instance metatable
	if ud := val.AsUserdata(); ud != nil {
		if mt := ud.Metatable(); mt != nil {
			// Check for __metatable field - if present, return that instead
			protected := mt.Get(vm.NewString("__metatable"))
			if !protected.IsNil() {
				v.Set(0, protected)
				return 1
			}
			v.Set(0, vm.NewTable(mt))
			return 1
		}
		v.Set(0, vm.Nil)
		return 1
	}
	// Check type metatables for non-table types (and threads)
	if mt := v.GetTypeMeta(val); mt != nil {
		// Check for __metatable field
		protected := mt.Get(vm.NewString("__metatable"))
		if !protected.IsNil() {
			v.Set(0, protected)
			return 1
		}
		v.Set(0, vm.NewTable(mt))
		return 1
	}
	v.Set(0, vm.Nil)
	return 1
}

// setmetatable(table, metatable)
func luaSetmetatable(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		callerArgError(v, 1, "setmetatable", "table expected"+gotDesc(v, 1))
	}
	if v.ArgCount() < 2 {
		callerArgError(v, 2, "setmetatable", "nil or table expected, got no value")
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
		callerArgError(v, 2, "setmetatable", "nil or table expected"+gotDesc(v, 2))
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
	var mt vm.LuaTable
	if val.IsTable() {
		mt = val.AsTable().Metatable()
	} else if ud := val.AsUserdata(); ud != nil {
		mt = ud.Metatable()
	}
	if mt == nil {
		mt = v.GetTypeMeta(val)
	}
	if mt != nil {
		if ts := mt.Get(vm.NewString("__tostring")); !ts.IsNil() {
			exitNonYieldable := v.EnterNonYieldable()
			results, err := v.ProtectedCall(ts, []vm.Value{val})
			exitNonYieldable()
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
		// Check for __name metamethod (used when __tostring is absent)
		if mt := tbl.Metatable(); mt != nil {
			if name := mt.Get(vm.NewString("__name")); name.IsString() {
				return fmt.Sprintf("%s: %p", name.AsString(), tbl)
			}
		}
		return fmt.Sprintf("table: %p", tbl)
	case val.IsFunction():
		return fmt.Sprintf("function: %p", val.AsClosure())
	case val.IsNativeFunc():
		return "function: (native)"
	case val.IsUserdata():
		ud := val.AsUserdata()
		if mt := ud.Metatable(); mt != nil {
			if name := mt.Get(vm.NewString("__name")); name.IsString() {
				return fmt.Sprintf("%s: %p", name.AsString(), ud)
			}
		}
		return fmt.Sprintf("userdata: %p", ud)
	default:
		return "?"
	}
}

// parseIntWrapping parses a string in the given base with unsigned wrapping
// modulo 2^64, matching Lua 5.4's tonumber(s, base) overflow semantics.
func parseIntWrapping(s string, base int) (int64, bool) {
	if len(s) == 0 {
		return 0, false
	}

	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}

	if len(s) == 0 {
		return 0, false
	}

	var result uint64
	for _, c := range []byte(s) {
		var digit uint64
		switch {
		case c >= '0' && c <= '9':
			digit = uint64(c - '0')
		case c >= 'a' && c <= 'z':
			digit = uint64(c-'a') + 10
		case c >= 'A' && c <= 'Z':
			digit = uint64(c-'A') + 10
		default:
			return 0, false
		}
		if digit >= uint64(base) {
			return 0, false
		}
		result = result*uint64(base) + digit
	}

	if neg {
		return -int64(result), true
	}
	return int64(result), true
}
