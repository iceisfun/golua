package stdlib

import (
	"fmt"

	"github.com/iceisfun/golua/vm"
)

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
	got := val.Type()
	if v.ArgCount() < idx {
		got = "no value"
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (string expected, got %s)", idx, fname, got))
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
		panic(fmt.Sprintf("bad argument #%d to '%s' (number has no integer representation)", idx, fname))
	}
	got := val.Type()
	if v.ArgCount() < idx {
		got = "no value"
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (number expected, got %s)", idx, fname, got))
}

func posRelat(pos int64, len int) int {
	if pos >= 0 {
		return int(pos)
	}
	return len + int(pos) + 1
}
