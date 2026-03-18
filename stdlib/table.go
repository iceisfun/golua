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
}

// tableGetTable extracts a table from argument idx, panicking with a standard error if not a table.
const (
	tabR = 1 << iota
	tabW
	tabL
	tabRW = tabR | tabW
)

func tableCheckLike(v *vm.VM, idx int, fname string, need int) vm.Value {
	val := v.Get(idx)
	if val.IsTable() {
		return val
	}
	if ((need&tabR) == 0 || !v.GetMetafield(val, "__index").IsNil()) &&
		((need&tabW) == 0 || !v.GetMetafield(val, "__newindex").IsNil()) &&
		((need&tabL) == 0 || !v.GetMetafield(val, "__len").IsNil()) {
		return val
	}
	got := v.ObjTypeName(val)
	if v.ArgCount() < idx {
		got = "no value"
	}
	callerArgError(v, idx, fname, fmt.Sprintf("table expected, got %s", got))
	return vm.Nil // unreachable
}

// tableObjLen returns #val via metamethod-aware ObjLen, panicking on error.
func tableObjLen(v *vm.VM, val vm.Value) int {
	length, err := v.ObjLen(val)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "object length is not an integer") {
			// This error should include file:line prefix (added by recovery).
			panic(msg)
		}
		// Other length errors (e.g. "attempt to get length of a X value")
		// should not get file:line prefix from native context.
		panic(&vm.LuaError{Value: vm.NewString(msg)})
	}
	return length
}

// tableGetInt reads t[key] via metamethod-aware TableGetInt, panicking on error.
func tableGetIdx(v *vm.VM, obj vm.Value, key int) vm.Value {
	val, err := v.IndexValue(obj, vm.NewInt(int64(key)))
	if err != nil {
		panic(err)
	}
	return val
}

// tableSetIdx writes t[key]=value via metamethod-aware TableSetInt, panicking on error.
func tableSetIdx(v *vm.VM, obj vm.Value, key int, value vm.Value) {
	err := v.SetIndexInt(obj, key, value)
	if err != nil {
		panic(err)
	}
}

// table.concat(list [, sep [, i [, j]]])
func tableConcat(v *vm.VM) int {
	obj := tableCheckLike(v, 1, "table.concat", tabR|tabL)

	// Snapshot all args before any metamethod calls (__len, __index).
	// Metamethod frames can overlap arg slots on the stack.
	sep := ""
	if !v.Get(2).IsNil() {
		sep = getString(v, 2, "table.concat")
	}
	hasI := !v.Get(3).IsNil()
	hasJ := !v.Get(4).IsNil()
	var iArg, jArg int64
	if hasI {
		iArg = getInt(v, 3, "table.concat")
	}
	if hasJ {
		jArg = getInt(v, 4, "table.concat")
	}

	length := tableObjLen(v, v.Get(1))

	i := int64(1)
	if hasI {
		i = iArg
	}

	j := int64(length)
	if hasJ {
		j = jArg
	}

	var parts []string
	for idx := i; idx <= j; idx++ {
		val := tableGetIdx(v, obj, int(idx))
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
	obj := tableCheckLike(v, 1, "table.insert", tabRW|tabL)

	n := v.ArgCount()
	if n < 2 || n > 3 {
		name, _ := callerFuncName(v, "insert")
		panic(fmt.Sprintf("wrong number of arguments to '%s'", name))
	}
	length := tableObjLen(v, v.Get(1))

	if n == 2 {
		// table.insert(list, value) - append to end
		tableSetIdx(v, obj, length+1, v.Get(2))
	} else if n >= 3 {
		// table.insert(list, pos, value)
		pos := int(getInt(v, 2, "table.insert"))
		val := v.Get(3)

		if pos < 1 || pos-1 > length {
			callerArgError(v, 2, "table.insert", "position out of bounds")
		}

		// Shift elements up: use length+1 as the loop start (like C Lua)
		// so that signed overflow naturally skips the loop when
		// length == math.MaxInt64.
		for i := length + 1; i > pos; i-- {
			elem := tableGetIdx(v, obj, i-1)
			tableSetIdx(v, obj, i, elem)
		}
		tableSetIdx(v, obj, pos, val)
	}

	return 0
}

// table.remove(list [, pos])
// Lua 5.4 semantics: pos defaults to #list. If pos != #list, validate 1 <= pos <= #list.
func tableRemove(v *vm.VM) int {
	obj := tableCheckLike(v, 1, "table.remove", tabRW|tabL)

	length := tableObjLen(v, v.Get(1))
	pos := length
	if !v.Get(2).IsNil() {
		pos = int(getInt(v, 2, "table.remove"))
	}

	// Lua 5.4: validate only when pos != length (the default)
	// Valid range: 1 <= pos <= length+1
	if pos != length {
		if pos < 1 || pos-1 > length {
			callerArgError(v, 2, "table.remove", "position out of bounds")
		}
	}

	// pos beyond array (length+1 case or empty table): return nil, no modification
	if pos > length {
		v.Set(0, vm.Nil)
		return 1
	}

	// Get the value being removed
	removed := tableGetIdx(v, obj, pos)

	// Shift elements down
	for i := pos; i < length; i++ {
		elem := tableGetIdx(v, obj, i+1)
		tableSetIdx(v, obj, i, elem)
	}
	// Clear the last slot (or the removed slot when length == 0 and pos == 0)
	tableSetIdx(v, obj, length, vm.Nil)

	v.Set(0, removed)
	return 1
}

// table.sort(list [, comp])
// Sorts in-place through metamethods, matching Lua 5.4's auxsort behavior.
func tableSort(v *vm.VM) int {
	obj := tableCheckLike(v, 1, "table.sort", tabRW|tabL)

	length := tableObjLen(v, v.Get(1))
	if length <= 1 {
		return 0
	}

	// Guard against absurd lengths from __len metamethod
	const maxSortLen = 1 << 30
	if length > maxSortLen {
		panic("bad argument #1 to 'table.sort' (array too big)")
	}

	comp := v.Get(2)
	var sortErr any

	if comp.IsNil() {
		auxSort(v, obj, 1, length, vm.Nil, &sortErr)
	} else {
		if !comp.IsFunction() && !comp.IsNativeFunc() {
			callerArgError(v, 2, "table.sort", fmt.Sprintf("function expected, got %s", v.ObjTypeName(comp)))
		}
		auxSort(v, obj, 1, length, comp, &sortErr)
	}
	if sortErr != nil {
		if val, errIsValue := sortErr.(vm.Value); errIsValue {
			panic(&vm.LuaError{Value: val})
		}
		if le, isLuaErr := sortErr.(*vm.LuaError); isLuaErr {
			panic(le)
		}
		if err, isErr := sortErr.(error); isErr {
			panic(err.Error())
		}
		panic(sortErr)
	}

	return 0
}

// sortGet reads table element at 1-based index through metamethods.
func sortGet(v *vm.VM, obj vm.Value, idx int, err *any) vm.Value {
	if *err != nil {
		return vm.Nil
	}
	return tableGetIdx(v, obj, idx)
}

// sortSet writes table element at 1-based index through metamethods.
func sortSet(v *vm.VM, obj vm.Value, idx int, val vm.Value, err *any) {
	if *err != nil {
		return
	}
	tableSetIdx(v, obj, idx, val)
}

// sortSwap swaps two elements in the table through metamethods.
func sortSwap(v *vm.VM, obj vm.Value, i, j int, err *any) {
	if *err != nil {
		return
	}
	a := tableGetIdx(v, obj, i)
	b := tableGetIdx(v, obj, j)
	tableSetIdx(v, obj, i, b)
	tableSetIdx(v, obj, j, a)
}

// sortComp evaluates a < b using the optional user comparator or CompareLT.
func sortComp(v *vm.VM, a, b vm.Value, comp vm.Value, err *any) bool {
	if *err != nil {
		return false
	}
	if comp == vm.Nil || comp.IsNil() {
		lt, e := v.CompareLT(a, b)
		if e != nil {
			// Wrap as LuaError to avoid AddCallerLocation adding a file:line
			// prefix. Comparison errors originate from native/C-level code
			// (luaG_runerror equivalent), not from Lua bytecode.
			*err = &vm.LuaError{Value: vm.NewString(e.Error())}
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

// auxSort implements Lua 5.4's sorting algorithm (QuickSort with median-of-3 pivot).
// Sorts in-place on the table through metamethods (__index/__newindex).
// lo and up are 1-based indices into the table.
func auxSort(v *vm.VM, obj vm.Value, lo, up int, comp vm.Value, err *any) {
	for lo < up {
		// Read and compare tbl[lo] and tbl[up]; handles 2-element case
		loVal := sortGet(v, obj, lo, err)
		upVal := sortGet(v, obj, up, err)
		if sortComp(v, upVal, loVal, comp, err) {
			sortSet(v, obj, lo, upVal, err)
			sortSet(v, obj, up, loVal, err)
		}
		if *err != nil {
			return
		}
		if up-lo == 1 {
			return
		}

		p := (lo + up) / 2

		// Median-of-3: sort tbl[lo], tbl[p], tbl[up] to pick pivot
		pVal := sortGet(v, obj, p, err)
		loVal = sortGet(v, obj, lo, err)
		if sortComp(v, pVal, loVal, comp, err) {
			sortSet(v, obj, p, loVal, err)
			sortSet(v, obj, lo, pVal, err)
		} else {
			upVal = sortGet(v, obj, up, err)
			if sortComp(v, upVal, pVal, comp, err) {
				sortSet(v, obj, up, pVal, err)
				sortSet(v, obj, p, upVal, err)
			}
		}
		if *err != nil {
			return
		}
		if up-lo == 2 {
			return
		}

		// Pivot is tbl[p]; move it to tbl[up-1]
		pivot := sortGet(v, obj, p, err)
		upM1Val := sortGet(v, obj, up-1, err)
		sortSet(v, obj, p, upM1Val, err)
		sortSet(v, obj, up-1, pivot, err)

		// Partition: match Lua 5.4's partition() exactly
		i := lo
		j := up - 1
		for {
			// Scan right: find first tbl[i] >= pivot
			for {
				i++
				iVal := sortGet(v, obj, i, err)
				if !sortComp(v, iVal, pivot, comp, err) {
					break
				}
				if i == up-1 {
					*err = fmt.Errorf("invalid order function for sorting")
					return
				}
			}
			// Scan left: find first tbl[j] <= pivot
			for {
				j--
				jVal := sortGet(v, obj, j, err)
				if !sortComp(v, pivot, jVal, comp, err) {
					break
				}
				if j < i {
					*err = fmt.Errorf("invalid order function for sorting")
					return
				}
			}
			if *err != nil {
				return
			}
			if j < i {
				// Swap pivot back into position
				iVal := sortGet(v, obj, i, err)
				sortSet(v, obj, up-1, iVal, err)
				sortSet(v, obj, i, pivot, err)
				break
			}
			// Swap tbl[i] and tbl[j]
			sortSwap(v, obj, i, j, err)
		}

		// Recurse on smaller partition, tail-call on larger
		if i-lo < up-i {
			auxSort(v, obj, lo, i-1, comp, err)
			lo = i + 1
		} else {
			auxSort(v, obj, i+1, up, comp, err)
			up = i - 1
		}
	}
}

// table.unpack(list [, i [, j]])
func tableUnpack(v *vm.VM) int {
	list := v.Get(1)

	// Snapshot optional args before __len metamethod can clobber slots.
	hasI := !v.Get(2).IsNil()
	hasJ := !v.Get(3).IsNil()
	var iArg, jArg int64
	if hasI {
		iArg = getInt(v, 2, "table.unpack")
	}
	if hasJ {
		jArg = getInt(v, 3, "table.unpack")
	}

	length := tableObjLen(v, list)

	i := int64(1)
	if hasI {
		i = iArg
	}

	j := int64(length)
	if hasJ {
		j = jArg
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

	// Snapshot all values into a Go slice first, then write to stack.
	// Writing results incrementally via v.Set() while calling __index
	// metamethods between writes would clobber earlier results because
	// metamethod frames overlap the result slots on the stack.
	results := make([]vm.Value, 0, nResults)
	for idx := i; idx <= j; idx++ {
		val := tableGetIdx(v, list, int(idx))
		results = append(results, val)
		if idx == j {
			break // avoid int64 overflow on idx++ when j == math.MaxInt64
		}
	}
	for k, val := range results {
		v.Set(k, val)
	}
	return len(results)
}

// table.pack(...)
func tablePack(v *vm.VM) int {
	n := v.ArgCount()
	tbl := vm.NewEmptyTable()
	tbl.EnsureArraySize(n)

	for i := 1; i <= n; i++ {
		tbl.RawSetArray(i, v.Get(i))
	}
	tbl.ShrinkArray()
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

	var a1, a2 vm.Value
	if !v.Get(5).IsNil() {
		a1 = tableCheckLike(v, 1, "table.move", tabR)
		a2 = tableCheckLike(v, 5, "table.move", tabW)
	} else {
		a1 = tableCheckLike(v, 1, "table.move", tabRW)
		a2 = a1
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

		if a1.RawEqual(a2) && tt > f && tt <= e {
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

	v.Set(0, a2)
	return 1
}
