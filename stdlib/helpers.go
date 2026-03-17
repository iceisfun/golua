package stdlib

import (
	"fmt"

	"github.com/iceisfun/golua/vm"
)

// callerFuncName resolves the native function's name from the calling Lua
// frame's bytecode. Returns the resolved name and nameWhat. If resolution
// fails, returns (fallback, "").
func callerFuncName(v *vm.VM, fallback string) (name, nameWhat string) {
	name, nameWhat = v.CallerFuncName()
	if name == "" {
		return fallback, ""
	}
	return name, nameWhat
}

// callerArgError panics with a "bad argument" error, resolving the function
// name from the calling Lua frame's bytecode debug info (matching Lua 5.4's
// luaL_argerror behavior). When the function was called via method syntax
// (e.g., s:format()), the argument index is decremented by 1 so that the
// implicit self parameter is not counted.
// The fallback name is used when bytecode resolution fails.
func callerArgError(v *vm.VM, idx int, fallback, msg string) {
	name, nameWhat := v.CallerFuncName()
	if name == "" {
		name = fallback
	}
	if nameWhat == "method" {
		if idx == 1 {
			// Lua 5.4's luaL_typeerror: when arg #1 was passed as self
			// via method syntax (OP_SELF), produce the special message.
			panic(fmt.Sprintf("calling '%s' on bad self (%s)", name, msg))
		}
		if idx > 0 {
			idx--
		}
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (%s)", idx, name, msg))
}

// getString returns the string value at stack index idx, coercing numbers.
// Panics with a Lua error message on type mismatch. This panic is always
// caught by ProtectedCall — see package-level panic convention docs.
func getString(v *vm.VM, idx int, fname string) string {
	val := v.Get(idx)
	if val.IsString() {
		return val.AsString()
	}
	if val.IsNumber() {
		return val.String()
	}
	got := v.ObjTypeName(val)
	if v.ArgCount() < idx {
		got = "no value"
	}
	callerArgError(v, idx, fname, fmt.Sprintf("string expected, got %s", got))
	return "" // unreachable
}

// getInt returns the integer value at stack index idx.
// Panics with a Lua error message on type mismatch. This panic is always
// caught by ProtectedCall — see package-level panic convention docs.
func getInt(v *vm.VM, idx int, fname string) int64 {
	val := v.Get(idx)
	if i, ok := val.ToInt(); ok {
		return i
	}
	if val.IsNumber() {
		callerArgError(v, idx, fname, "number has no integer representation")
	}
	got := v.ObjTypeName(val)
	if v.ArgCount() < idx {
		got = "no value"
	}
	callerArgError(v, idx, fname, fmt.Sprintf("number expected, got %s", got))
	return 0 // unreachable
}

// getNumber returns the float64 value at stack index idx, coercing integers.
// Panics with a Lua error message on type mismatch. This panic is always
// caught by ProtectedCall — see package-level panic convention docs.
func getNumber(v *vm.VM, idx int, fname string) float64 {
	val := v.Get(idx)
	if n, ok := val.ToNumber(); ok {
		return n
	}
	got := v.ObjTypeName(val)
	if v.ArgCount() < idx {
		got = "no value"
	}
	callerArgError(v, idx, fname, fmt.Sprintf("number expected, got %s", got))
	return 0 // unreachable
}

func posRelat(pos int64, len int) int {
	if pos >= 0 {
		return int(pos)
	}
	return len + int(pos) + 1
}
