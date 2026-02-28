package stdlib

import (
	"fmt"

	"github.com/iceisfun/golua/vm"
)

func getString(v *vm.VM, idx int, fname string) string {
	val := v.Get(idx)
	if val.IsString() {
		return val.AsString()
	}
	if val.IsNumber() {
		return val.String()
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (string expected, got %s)", idx, fname, val.Type()))
}

func getInt(v *vm.VM, idx int, fname string) int64 {
	val := v.Get(idx)
	if i, ok := val.ToInt(); ok {
		return i
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (number expected, got %s)", idx, fname, val.Type()))
}

func posRelat(pos int64, len int) int {
	if pos >= 0 {
		return int(pos)
	}
	return len + int(pos) + 1
}
