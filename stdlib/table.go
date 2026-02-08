package stdlib

import (
	"sort"
	"strings"

	"github.com/iceisfun/golua/vm"
)

func openTable(v *vm.VM) {
	t := vm.NewEmptyTable()

	t.SetString("concat", vm.NewNativeFunc(tableConcat))
	t.SetString("insert", vm.NewNativeFunc(tableInsert))
	t.SetString("remove", vm.NewNativeFunc(tableRemove))
	t.SetString("sort", vm.NewNativeFunc(tableSort))
	t.SetString("unpack", vm.NewNativeFunc(tableUnpack))
	t.SetString("pack", vm.NewNativeFunc(tablePack))
	t.SetString("move", vm.NewNativeFunc(tableMove))

	v.SetGlobal("table", vm.NewTable(t))

	// Also register unpack as a global (for compatibility)
	v.SetGlobal("unpack", vm.NewNativeFunc(tableUnpack))
}

// table.concat(list [, sep [, i [, j]]])
func tableConcat(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'concat' (table expected)")
	}

	sep := ""
	if !v.Get(2).IsNil() {
		sep = getString(v, 2, "concat")
	}

	i := int64(1)
	if !v.Get(3).IsNil() {
		i = getInt(v, 3, "concat")
	}

	j := int64(tbl.Len())
	if !v.Get(4).IsNil() {
		j = getInt(v, 4, "concat")
	}

	var parts []string
	for idx := i; idx <= j; idx++ {
		val := tbl.Get(vm.NewInt(idx))
		if val.IsString() {
			parts = append(parts, val.AsString())
		} else if val.IsNumber() {
			parts = append(parts, val.String())
		} else {
			panic("invalid value (nil) at index " + val.String() + " in table for 'concat'")
		}
	}

	v.Set(0, vm.NewString(strings.Join(parts, sep)))
	return 1
}

// table.insert(list, [pos,] value)
func tableInsert(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'insert' (table expected)")
	}

	n := v.ArgCount()
	length := tbl.Len()

	if n == 2 {
		// table.insert(list, value) - append to end
		tbl.Set(vm.NewInt(int64(length+1)), v.Get(2))
	} else if n >= 3 {
		// table.insert(list, pos, value)
		pos := int(getInt(v, 2, "insert"))
		val := v.Get(3)

		// Shift elements up
		for i := length; i >= pos; i-- {
			tbl.Set(vm.NewInt(int64(i+1)), tbl.Get(vm.NewInt(int64(i))))
		}
		tbl.Set(vm.NewInt(int64(pos)), val)
	}

	return 0
}

// table.remove(list [, pos])
func tableRemove(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'remove' (table expected)")
	}

	length := tbl.Len()
	pos := length
	if !v.Get(2).IsNil() {
		pos = int(getInt(v, 2, "remove"))
	}

	if pos < 1 || pos > length {
		v.Set(0, vm.Nil)
		return 1
	}

	removed := tbl.Get(vm.NewInt(int64(pos)))

	// Shift elements down
	for i := pos; i < length; i++ {
		tbl.Set(vm.NewInt(int64(i)), tbl.Get(vm.NewInt(int64(i+1))))
	}
	tbl.Set(vm.NewInt(int64(length)), vm.Nil)

	v.Set(0, removed)
	return 1
}

// table.sort(list [, comp])
func tableSort(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'sort' (table expected)")
	}

	length := tbl.Len()
	if length <= 1 {
		return 0
	}

	// Extract values into slice
	values := make([]vm.Value, length)
	for i := 1; i <= length; i++ {
		values[i-1] = tbl.Get(vm.NewInt(int64(i)))
	}

	comp := v.Get(2)

	if comp.IsNil() {
		// Default comparison: a < b
		sort.Slice(values, func(i, j int) bool {
			lt, ok := values[i].LessThan(values[j])
			return ok && lt
		})
	} else {
		// TODO: Implement custom comparator
		// For now, just use default
		sort.Slice(values, func(i, j int) bool {
			lt, ok := values[i].LessThan(values[j])
			return ok && lt
		})
	}

	// Put values back
	for i := 1; i <= length; i++ {
		tbl.Set(vm.NewInt(int64(i)), values[i-1])
	}

	return 0
}

// table.unpack(list [, i [, j]])
func tableUnpack(v *vm.VM) int {
	tbl := v.Get(1).AsTable()
	if tbl == nil {
		panic("bad argument #1 to 'unpack' (table expected)")
	}

	i := int64(1)
	if !v.Get(2).IsNil() {
		i = getInt(v, 2, "unpack")
	}

	j := int64(tbl.Len())
	if !v.Get(3).IsNil() {
		j = getInt(v, 3, "unpack")
	}

	count := 0
	for idx := i; idx <= j; idx++ {
		v.Set(count, tbl.Get(vm.NewInt(idx)))
		count++
	}
	return count
}

// table.pack(...)
func tablePack(v *vm.VM) int {
	n := v.ArgCount()
	tbl := vm.NewEmptyTable()

	for i := 1; i <= n; i++ {
		tbl.SetInt(i, v.Get(i))
	}
	tbl.SetString("n", vm.NewInt(int64(n)))

	v.Set(0, vm.NewTable(tbl))
	return 1
}

// table.move(a1, f, e, t [,a2])
func tableMove(v *vm.VM) int {
	a1 := v.Get(1).AsTable()
	if a1 == nil {
		panic("bad argument #1 to 'move' (table expected)")
	}

	f := int(getInt(v, 2, "move"))
	e := int(getInt(v, 3, "move"))
	t := int(getInt(v, 4, "move"))

	a2 := a1
	if !v.Get(5).IsNil() {
		a2 = v.Get(5).AsTable()
		if a2 == nil {
			panic("bad argument #5 to 'move' (table expected)")
		}
	}

	if f <= e {
		count := e - f + 1
		if t > f {
			// Copy backwards to avoid overwriting
			for i := count - 1; i >= 0; i-- {
				a2.Set(vm.NewInt(int64(t+i)), a1.Get(vm.NewInt(int64(f+i))))
			}
		} else {
			for i := 0; i < count; i++ {
				a2.Set(vm.NewInt(int64(t+i)), a1.Get(vm.NewInt(int64(f+i))))
			}
		}
	}

	v.Set(0, vm.NewTable(a2))
	return 1
}
