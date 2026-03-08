package stdlib

import (
	"fmt"
	"math"
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
	got := val.Type()
	if v.ArgCount() < idx {
		got = "no value"
	}
	callerArgError(v, idx, fname, fmt.Sprintf("table expected, got %s", got))
	return nil // unreachable
}

// tableObjLen returns #val via metamethod-aware ObjLen, panicking on error.
func tableObjLen(v *vm.VM, val vm.Value) int {
	length, err := v.ObjLen(val)
	if err != nil {
		panic(err)
	}
	return length
}

// tableGetInt reads t[key] via metamethod-aware TableGetInt, panicking on error.
func tableGetIdx(v *vm.VM, t vm.LuaTable, key int) vm.Value {
	val, err := v.TableGetInt(t, key)
	if err != nil {
		panic(err)
	}
	return val
}

// tableSetIdx writes t[key]=value via metamethod-aware TableSetInt, panicking on error.
func tableSetIdx(v *vm.VM, t vm.LuaTable, key int, value vm.Value) {
	err := v.TableSetInt(t, key, value)
	if err != nil {
		panic(err)
	}
}

// table.concat(list [, sep [, i [, j]]])
func tableConcat(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "table.concat")

	sep := ""
	if !v.Get(2).IsNil() {
		sep = getString(v, 2, "table.concat")
	}

	length := tableObjLen(v, v.Get(1))

	i := int64(1)
	if !v.Get(3).IsNil() {
		i = getInt(v, 3, "table.concat")
	}

	j := int64(length)
	if !v.Get(4).IsNil() {
		j = getInt(v, 4, "table.concat")
	}

	var parts []string
	for idx := i; idx <= j; idx++ {
		val := tableGetIdx(v, tbl, int(idx))
		if val.IsString() {
			parts = append(parts, val.AsString())
		} else if val.IsNumber() {
			parts = append(parts, val.String())
		} else {
			panic(fmt.Sprintf("invalid value (%s) at index %d in table for 'concat'", val.Type(), idx))
		}
		if idx == j {
			break // avoid int64 overflow on idx++
		}
	}

	v.Set(0, vm.NewString(strings.Join(parts, sep)))
	return 1
}

// table.insert(list, [pos,] value)
func tableInsert(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "table.insert")

	n := v.ArgCount()
	if n < 2 || n > 3 {
		name, _ := callerFuncName(v, "insert")
		panic(fmt.Sprintf("wrong number of arguments to '%s'", name))
	}
	length := tableObjLen(v, v.Get(1))

	if n == 2 {
		// table.insert(list, value) - append to end
		tableSetIdx(v, tbl, length+1, v.Get(2))
	} else if n >= 3 {
		// table.insert(list, pos, value)
		pos := int(getInt(v, 2, "table.insert"))
		val := v.Get(3)

		if pos < 1 || pos > length+1 {
			callerArgError(v, 2, "table.insert", "position out of bounds")
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
	tbl := tableGetTable(v, 1, "table.remove")

	length := tableObjLen(v, v.Get(1))
	pos := length
	if !v.Get(2).IsNil() {
		pos = int(getInt(v, 2, "table.remove"))
	}

	// Lua 5.4: validate only when pos != length (the default)
	// Valid range: 1 <= pos <= length+1
	if pos != length {
		if pos < 1 || pos > length+1 {
			callerArgError(v, 2, "table.remove", "position out of bounds")
		}
	}

	// pos beyond array (length+1 case or empty table): return nil, no modification
	if pos > length {
		v.Set(0, vm.Nil)
		return 1
	}

	// Get the value being removed
	removed := tableGetIdx(v, tbl, pos)

	// Shift elements down
	for i := pos; i < length; i++ {
		elem := tableGetIdx(v, tbl, i+1)
		tableSetIdx(v, tbl, i, elem)
	}
	// Clear the last slot (or the removed slot when length == 0 and pos == 0)
	tableSetIdx(v, tbl, length, vm.Nil)

	v.Set(0, removed)
	return 1
}

// table.sort(list [, comp])
func tableSort(v *vm.VM) int {
	tbl := tableGetTable(v, 1, "table.sort")

	length := tableObjLen(v, v.Get(1))
	if length <= 1 {
		return 0
	}

	// Guard against absurd lengths from __len metamethod
	const maxSortLen = 1 << 30
	if length > maxSortLen {
		panic("bad argument #1 to 'table.sort' (array too big)")
	}

	// Extract values into slice via __index
	values := make([]vm.Value, length)
	for i := 1; i <= length; i++ {
		values[i-1] = tableGetIdx(v, tbl, i)
	}

	comp := v.Get(2)
	var sortErr any

	if comp.IsNil() {
		// Default comparison: a < b (via metamethod-aware CompareLT)
		auxSort(v, values, 0, length-1, vm.Nil, &sortErr)
	} else {
		if !comp.IsFunction() && !comp.IsNativeFunc() {
			callerArgError(v, 2, "table.sort", fmt.Sprintf("function expected, got %s", comp.Type()))
		}
		// Custom comparator
		auxSort(v, values, 0, length-1, comp, &sortErr)
	}
	if sortErr != nil {
		if val, errIsValue := sortErr.(vm.Value); errIsValue {
			panic(&vm.LuaError{Value: val})
		}
		if err, isErr := sortErr.(error); isErr {
			panic(err.Error())
		}
		panic(sortErr)
	}

	// Put values back via __newindex
	for i := 1; i <= length; i++ {
		tableSetIdx(v, tbl, i, values[i-1])
	}

	return 0
}

// sortComp evaluates a < b using the optional user comparator or CompareLT.
func sortComp(v *vm.VM, a, b vm.Value, comp vm.Value, err *any) bool {
	if *err != nil {
		return false
	}
	if comp == vm.Nil || comp.IsNil() {
		lt, e := v.CompareLT(a, b)
		if e != nil {
			*err = e
			return false
		}
		return lt
	}
	exitNonYieldable := v.EnterNonYieldable()
	defer exitNonYieldable()
	res, e := v.ProtectedCall(comp, []vm.Value{a, b})
	if e != nil {
		if luaErr, ok := e.(*vm.LuaError); ok {
			*err = luaErr.Value
		} else {
			*err = e
		}
		return false
	}
	if len(res) == 0 {
		return false
	}
	return res[0].ToBool()
}

// auxSort implements Lua 5.4's sorting algorithm (QuickSort with InsertionSort fallback).
func auxSort(v *vm.VM, a []vm.Value, lo, up int, comp vm.Value, err *any) {
	for lo < up {
		p := (lo + up) / 2
		// Sort elements a[lo], a[p], a[up]
		if sortComp(v, a[up], a[p], comp, err) {
			a[up], a[p] = a[p], a[up]
		}
		if sortComp(v, a[p], a[lo], comp, err) {
			a[p], a[lo] = a[lo], a[p]
		}
		if sortComp(v, a[up], a[p], comp, err) {
			a[up], a[p] = a[p], a[up]
		}
		if *err != nil {
			return
		}

		if up-lo == 1 {
			return
		}
		if up-lo == 2 {
			if sortComp(v, a[up], a[p], comp, err) {
				a[up], a[p] = a[p], a[up]
			}
			return
		}

		// Pivot is a[p]; validate invariant: !(pivot < a[lo])
		pivot := a[p]
		if sortComp(v, pivot, a[lo], comp, err) {
			*err = fmt.Errorf("invalid order function for sorting")
			return
		}
		if *err != nil {
			return
		}
		a[p], a[up-1] = a[up-1], a[p]

		i := lo
		j := up - 1

		for {
			for {
				i++
				if !sortComp(v, a[i], pivot, comp, err) {
					break
				}
				if i >= up-1 {
					*err = fmt.Errorf("invalid order function for sorting")
					return
				}
			}
			for {
				j--
				if !sortComp(v, pivot, a[j], comp, err) {
					break
				}
				if j <= lo {
					*err = fmt.Errorf("invalid order function for sorting")
					return
				}
			}
			if *err != nil {
				return
			}
			if i >= j {
				break
			}
			a[i], a[j] = a[j], a[i]
		}
		// Validate invariant: !(a[up] < pivot)
		if sortComp(v, a[up], pivot, comp, err) {
			*err = fmt.Errorf("invalid order function for sorting")
			return
		}
		if *err != nil {
			return
		}
		a[up-1], a[i] = a[i], a[up-1]

		if i-lo < up-i {
			auxSort(v, a, lo, i-1, comp, err)
			lo = i + 1
		} else {
			auxSort(v, a, i+1, up, comp, err)
			up = i - 1
		}
	}
}

// table.unpack(list [, i [, j]])
func tableUnpack(v *vm.VM) int {
	list := v.Get(1)

	length := tableObjLen(v, list)

	i := int64(1)
	if !v.Get(2).IsNil() {
		i = getInt(v, 2, "table.unpack")
	}

	j := int64(length)
	if !v.Get(3).IsNil() {
		j = getInt(v, 3, "table.unpack")
	}

	if j < i {
		return 0
	}
	// Compute result count carefully to avoid overflow.
	n := uint64(j) - uint64(i) + 1
	if n == 0 || n >= math.MaxInt32 {
		panic("too many results to unpack")
	}
	nResults := int(n)
	// Check stack capacity before allocating (matches Lua 5.4's luaL_checkstack)
	if nResults > 0 {
		if !v.CheckStack(v.Base() + nResults) {
			panic("too many results to unpack")
		}
		v.EnsureStack(v.Base() + nResults)
	}

	// Use table-optimized path when possible, generic index otherwise
	tbl := list.AsTable()
	k := 0
	for idx := i; idx <= j; idx++ {
		var val vm.Value
		if tbl != nil {
			val = tableGetIdx(v, tbl, int(idx))
		} else {
			var err error
			val, err = v.IndexValue(list, vm.NewInt(idx))
			if err != nil {
				panic(err)
			}
		}
		v.Set(k, val)
		k++
		if idx == j {
			break // avoid int64 overflow on idx++ when j == math.MaxInt64
		}
	}
	return k
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
	// Check numeric args before table args (matches Lua 5.4 luaB_move order)
	f := getInt(v, 2, "table.move")
	e := getInt(v, 3, "table.move")
	tt := getInt(v, 4, "table.move")

	a1 := tableGetTable(v, 1, "table.move")

	a2 := a1
	if !v.Get(5).IsNil() {
		a2 = tableGetTable(v, 5, "table.move")
	}

	if f <= e {
		// Check interval too large
		if f < 0 && e > 0 && e-f < 0 {
			// overflow in e-f
			callerArgError(v, 3, "table.move", "too many elements to move")
		}
		count := e - f + 1
		if count < 0 {
			callerArgError(v, 3, "table.move", "too many elements to move")
		}

		// Check destination overflow: tt + (e - f) must not wrap
		if e-f >= 0 {
			dest_end := tt + (e - f)
			if (e-f > 0) && (dest_end < tt) {
				callerArgError(v, 4, "table.move", "destination wrap around")
			}
			// Also check against math.MaxInt64
			if tt > 0 && e-f > math.MaxInt64-tt {
				callerArgError(v, 4, "table.move", "destination wrap around")
			}
		}

		if a1 == a2 && tt > f && tt <= e {
			// Copy backwards to avoid overwriting (overlapping same-table move)
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
