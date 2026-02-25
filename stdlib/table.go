package stdlib

import (
	"fmt"
	"math"
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

// tableGetTable extracts a table from argument idx, panicking with a standard error if not a table.
func tableGetTable(v *vm.VM, idx int, fname string) vm.LuaTable {
	val := v.Get(idx)
	if val.IsTable() {
		return val.AsTable()
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (table expected, got %s)", idx, fname, val.Type()))
}

// tableObjLen returns #val via metamethod-aware ObjLen, panicking on error.
func tableObjLen(v *vm.VM, val vm.Value) int {
	length, err := v.ObjLen(val)
	if err != nil {
		panic(err.Error())
	}
	return length
}

// tableGetInt reads t[key] via metamethod-aware TableGetInt, panicking on error.
func tableGetIdx(v *vm.VM, t vm.LuaTable, key int) vm.Value {
	val, err := v.TableGetInt(t, key)
	if err != nil {
		panic(err.Error())
	}
	return val
}

// tableSetIdx writes t[key]=value via metamethod-aware TableSetInt, panicking on error.
func tableSetIdx(v *vm.VM, t vm.LuaTable, key int, value vm.Value) {
	err := v.TableSetInt(t, key, value)
	if err != nil {
		panic(err.Error())
	}
}

// table.concat(list [, sep [, i [, j]]])
func tableConcat(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "concat")

	sep := ""
	if !v.Get(2).IsNil() {
		sep = getString(v, 2, "concat")
	}

	length := tableObjLen(v, v.Get(1))

	i := int64(1)
	if !v.Get(3).IsNil() {
		i = getInt(v, 3, "concat")
	}

	j := int64(length)
	if !v.Get(4).IsNil() {
		j = getInt(v, 4, "concat")
	}

	var parts []string
	for idx := i; idx <= j; idx++ {
		val := tableGetIdx(v, tbl, int(idx))
		if val.IsString() {
			parts = append(parts, val.AsString())
		} else if val.IsNumber() {
			parts = append(parts, val.String())
		} else {
			panic(fmt.Sprintf("invalid value (nil) at index %d in table for 'concat'", idx))
		}
	}

	v.Set(0, vm.NewString(strings.Join(parts, sep)))
	return 1
}

// table.insert(list, [pos,] value)
func tableInsert(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "insert")

	n := v.ArgCount()
	length := tableObjLen(v, v.Get(1))

	if n == 2 {
		// table.insert(list, value) - append to end
		tableSetIdx(v, tbl, length+1, v.Get(2))
	} else if n >= 3 {
		// table.insert(list, pos, value)
		pos := int(getInt(v, 2, "insert"))
		val := v.Get(3)

		if pos < 1 || pos > length+1 {
			panic(fmt.Sprintf("bad argument #2 to 'insert' (position out of bounds)"))
		}

		// Shift elements up
		for i := length; i >= pos; i-- {
			elem := tableGetIdx(v, tbl, i)
			tableSetIdx(v, tbl, i+1, elem)
		}
		tableSetIdx(v, tbl, pos, val)
	}

	return 0
}

// table.remove(list [, pos])
// Lua 5.4 semantics: pos defaults to #list. If pos != #list, validate 1 <= pos <= #list.
func tableRemove(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "remove")

	length := tableObjLen(v, v.Get(1))
	pos := length
	if !v.Get(2).IsNil() {
		pos = int(getInt(v, 2, "remove"))
	}

	// Lua 5.4: validate only when pos != length
	if pos != length {
		if pos < 1 || pos > length {
			panic("bad argument #2 to 'remove' (position out of range)")
		}
	}

	// Get the value being removed
	removed := tableGetIdx(v, tbl, pos)

	// Shift elements down
	for i := pos; i < length; i++ {
		elem := tableGetIdx(v, tbl, i+1)
		tableSetIdx(v, tbl, i, elem)
	}
	if length > 0 {
		tableSetIdx(v, tbl, length, vm.Nil)
	}

	v.Set(0, removed)
	return 1
}

// table.sort(list [, comp])
func tableSort(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "sort")

	length := tableObjLen(v, v.Get(1))
	if length <= 1 {
		return 0
	}

	// Extract values into slice via __index
	values := make([]vm.Value, length)
	for i := 1; i <= length; i++ {
		values[i-1] = tableGetIdx(v, tbl, i)
	}

	comp := v.Get(2)
	var sortErr error

	if comp.IsNil() {
		// Default comparison: a < b
		sort.Slice(values, func(i, j int) bool {
			if sortErr != nil {
				return false
			}
			lt, ok := values[i].LessThan(values[j])
			if !ok {
				sortErr = fmt.Errorf("attempt to compare %s with %s", values[i].Type(), values[j].Type())
				return false
			}
			return lt
		})
	} else {
		if !comp.IsFunction() && !comp.IsNativeFunc() {
			panic(fmt.Sprintf("bad argument #2 to 'sort' (function expected, got %s)", comp.Type()))
		}
		// Custom comparator: call the Lua function for each comparison
		sort.Slice(values, func(i, j int) bool {
			if sortErr != nil {
				return false // abort: don't reorder after error
			}
			results, err := v.ProtectedCall(comp, []vm.Value{values[i], values[j]})
			if err != nil {
				sortErr = err
				return false
			}
			if len(results) == 0 {
				return false
			}
			return results[0].ToBool()
		})
	}
	if sortErr != nil {
		panic(sortErr.Error())
	}

	// Put values back via __newindex
	for i := 1; i <= length; i++ {
		tableSetIdx(v, tbl, i, values[i-1])
	}

	return 0
}

// table.unpack(list [, i [, j]])
func tableUnpack(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "unpack")

	length := tableObjLen(v, v.Get(1))

	i := int64(1)
	if !v.Get(2).IsNil() {
		i = getInt(v, 2, "unpack")
	}

	j := int64(length)
	if !v.Get(3).IsNil() {
		j = getInt(v, 3, "unpack")
	}

	if j < i {
		return 0
	}
	// Match Lua behavior: reject pathological ranges that would produce an
	// impractically large number of return values.
	if i < 0 && j > 0 && j-i < 0 {
		panic("too many results to unpack")
	}
	n64 := j - i + 1
	if n64 <= 0 || n64 > math.MaxInt32 {
		panic("too many results to unpack")
	}
	n := int(n64)
	if n > 0 {
		v.EnsureStack(v.Base() + n)
	}

	count := 0
	for idx := i; idx <= j; idx++ {
		val := tableGetIdx(v, tbl, int(idx))
		v.Set(count, val)
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
	a1 := tableGetTable(v, 1, "move")

	f := getInt(v, 2, "move")
	e := getInt(v, 3, "move")
	tt := getInt(v, 4, "move")

	a2 := a1
	if !v.Get(5).IsNil() {
		a2 = tableGetTable(v, 5, "move")
	}

	if f <= e {
		// Check interval too large
		if f < 0 && e > 0 && e-f < 0 {
			// overflow in e-f
			panic("too many elements to move (interval too large)")
		}
		count := e - f + 1
		if count < 0 {
			panic("too many elements to move (interval too large)")
		}

		// Check destination overflow: tt + (e - f) must not wrap
		if e-f >= 0 {
			dest_end := tt + (e - f)
			if (e-f > 0) && (dest_end < tt) {
				panic("destination wrap around")
			}
			// Also check against math.MaxInt64
			if tt > 0 && e-f > math.MaxInt64-tt {
				panic("destination wrap around")
			}
		}

		if tt > f {
			// Copy backwards to avoid overwriting
			for i := count - 1; i >= 0; i-- {
				val := tableGetIdx(v, a1, int(f+i))
				tableSetIdx(v, a2, int(tt+i), val)
			}
		} else {
			for i := int64(0); i < count; i++ {
				val := tableGetIdx(v, a1, int(f+i))
				tableSetIdx(v, a2, int(tt+i), val)
			}
		}
	}

	v.Set(0, vm.NewTable(a2))
	return 1
}
