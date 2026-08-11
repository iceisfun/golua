package stdlib

import (
	"fmt"
	"math"
	"strings"

	"github.com/iceisfun/golua/v2/vm"
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
	t.SetString("create", vm.NewNativeFunc(tableCreate))

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
	// Threads are backed by a *Table but must not pass as tables here
	// (reference checktab's lua_istable excludes LUA_TTHREAD); they can
	// still qualify via metafields below, like any other value.
	if val.IsTable() && !val.AsTable().IsThread() {
		return val
	}
	if ((need&tabR) == 0 || !v.GetMetafield(val, vm.MetaIndex).IsNil()) &&
		((need&tabW) == 0 || !v.GetMetafield(val, vm.MetaNewIndex).IsNil()) &&
		((need&tabL) == 0 || !v.GetMetafield(val, vm.MetaLen).IsNil()) {
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

// tableGetInt reads t[key] via metamethod-aware IndexInt, panicking on error.
func tableGetIdx(v *vm.VM, obj vm.Value, key int) vm.Value {
	val, err := v.IndexInt(obj, key)
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
	// C-level function: metamethods it triggers (__len, __index) must not yield
	// across this boundary, matching reference Lua ("attempt to yield across a
	// C-call boundary").
	defer v.EnterNonYieldable()()
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
	var total int64 // accumulated result size (incl. separators)
	for idx := i; idx <= j; idx++ {
		val := tableGetIdx(v, obj, int(idx))
		var part string
		if val.IsString() {
			part = val.AsString()
		} else if val.IsNumber() {
			part = val.String()
		} else {
			panic(fmt.Sprintf("invalid value (%s) at index %d in table for 'concat'", val.Type(), idx))
		}
		parts = append(parts, part)
		// Cap the accumulated result at the Go-safe limit string.rep/concat/gsub
		// use. Concatenating many large elements would otherwise build a result
		// past what Go can allocate and trigger an UNCATCHABLE runtime fatal OOM
		// that aborts the host (a sandbox escape); reject with a catchable error.
		total += int64(len(part)) + int64(len(sep))
		if total > maxStrResultSize {
			panic("resulting string too large")
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
	// C-level function: metamethods (__len, __index, __newindex) must not yield
	// across this boundary, matching reference Lua.
	defer v.EnterNonYieldable()()
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
	// C-level function: metamethods (__len, __index, __newindex) must not yield
	// across this boundary, matching reference Lua.
	defer v.EnterNonYieldable()()
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

	// Get the value being removed. Reference tremove does lua_geti/lua_seti
	// unconditionally, including for the allowed past-the-end position
	// (pos == #list+1), so the read and the final nil-out below stay
	// metamethod-aware there too.
	removed := tableGetIdx(v, obj, pos)

	// Shift elements down
	for i := pos; i < length; i++ {
		elem := tableGetIdx(v, obj, i+1)
		tableSetIdx(v, obj, i, elem)
	}
	// Clear the vacated slot: t[#list] after a shift, t[pos] itself when no
	// shift occurred (pos >= length, including pos == 0 on an empty table)
	clearPos := pos
	if length > clearPos {
		clearPos = length
	}
	tableSetIdx(v, obj, clearPos, vm.Nil)

	v.Set(0, removed)
	return 1
}

// table.sort(list [, comp])
// Sorts in-place through metamethods, matching Lua 5.4's auxsort behavior.
func tableSort(v *vm.VM) int {
	// C-level function: metamethods (__len, __index, __newindex, the default '<'
	// comparator's __lt) and the user comparator must not yield across this
	// boundary, matching reference Lua. This subsumes the comparator-call guard
	// inside sortComp.
	defer v.EnterNonYieldable()()
	obj := tableCheckLike(v, 1, "table.sort", tabRW|tabL)

	length := tableObjLen(v, v.Get(1))
	if length <= 1 {
		return 0
	}

	// Guard against absurd lengths from __len metamethod
	const maxSortLen = 1 << 30
	if length > maxSortLen {
		// Route through callerArgError like the sibling checks so the caller's
		// resolved name ('sort') is used instead of a hardcoded 'table.sort'.
		callerArgError(v, 1, "table.sort", "array too big")
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
	// The non-yieldable boundary is established once at tableSort entry.
	res, e := v.ProtectedCallNoTBCClose(comp, []vm.Value{a, b})
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
	// C-level function: metamethods (__len, __index) must not yield across this
	// boundary, matching reference Lua.
	defer v.EnterNonYieldable()()
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

	i := int64(1)
	if hasI {
		i = iArg
	}

	var j int64
	if hasJ {
		j = jArg
	} else {
		j = int64(tableObjLen(v, list))
	}

	if j < i {
		return 0
	}
	// Compute result count carefully to avoid overflow.
	n := uint64(j) - uint64(i) + 1
	if n == 0 || n > 1000000 {
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
	// C-level function: metamethods (__index, __newindex) must not yield across
	// this boundary, matching reference Lua.
	defer v.EnterNonYieldable()()
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
				moveCheckInterrupt(v, i)
				val := tableGetIdx(v, a1, int(f+i))
				tableSetIdx(v, a2, int(tt+i), val)
			}
		} else {
			for i := int64(0); i < count; i++ {
				moveCheckInterrupt(v, i)
				val := tableGetIdx(v, a1, int(f+i))
				tableSetIdx(v, a2, int(tt+i), val)
			}
		}
	}

	v.Set(0, a2)
	return 1
}

// moveCheckInterrupt polls for context cancellation and limit violations
// inside table.move's copy loop. Reference Lua accepts ranges spanning the
// whole integer domain (the official suite relies on it), so the range itself
// cannot be capped up front; without this poll such a move never re-enters the
// VM's instruction dispatch and ignores cancellation entirely.
func moveCheckInterrupt(v *vm.VM, i int64) {
	if i&0xff != 0 {
		return
	}
	if err := v.CheckInterrupt(); err != nil {
		panic(err)
	}
}

// maxTableCreateSize caps narr/nrec so `make([]...)` can never panic with
// "makeslice: cap out of range". Lua 5.5's reference implementation uses
// MAXASIZE (INT_MAX); for a sandboxed Go host any value this large would
// realistically OOM the process. 1<<30 (~1G entries) is absurdly large for
// a preallocation hint while leaving headroom before Go's own allocator
// rejects the request.
//
// The narr/nrec arguments are validated against the same limits as reference
// Lua 5.5, and crucially these limits bound only *acceptance*, not allocation:
//   - An argument exceeding INT_MAX no longer fits the C int the reference
//     uses, so it is rejected as the argument error "out of range".
//   - A hash hint (nrec) that fits in an int but exceeds the hash part's
//     MAXHSIZE raises the runtime error "table overflow" (no location prefix),
//     distinct from the argument "out of range".
//
// A large-but-accepted array hint (e.g. table.create(1<<29)) is NOT allocated
// eagerly: reference Lua only succeeds because Linux overcommit hands it a
// virtual mapping it never faults in, whereas Go's make zero-fills the whole
// backing store and would trigger a fatal, pcall-uncatchable out-of-memory
// abort. vm.NewTableWithSize independently clamps the real preallocation, so
// the table is returned (matching reference) without the giant allocation.
const (
	maxTableCreateArg  = math.MaxInt32 // INT_MAX: narr/nrec above this are "out of range"
	maxTableCreateHash = 1 << 30       // MAXHSIZE: nrec above this is "table overflow"
)

// table.create(narr [, nrec])
// Creates a new empty table with preallocated capacity for narr array slots
// and nrec hash slots. narr is required; nrec defaults to 0. Negative or
// excessively large values are rejected before any slice allocation so that
// no Go-internal allocation error can surface to Lua.
func tableCreate(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "table.create", "number expected, got no value")
	}
	narr := getInt(v, 1, "table.create")
	if narr < 0 || narr > maxTableCreateArg {
		callerArgError(v, 1, "table.create", "out of range")
	}

	var nrec int64
	if v.ArgCount() >= 2 && !v.Get(2).IsNil() {
		nrec = getInt(v, 2, "table.create")
		if nrec < 0 || nrec > maxTableCreateArg {
			callerArgError(v, 2, "table.create", "out of range")
		}
		if nrec > maxTableCreateHash {
			// Plain runtime error: reference Lua raises this without a
			// source-location prefix, so use a LuaError value (panic(string)
			// would gain a file:line prefix from the native-recovery layer).
			panic(&vm.LuaError{Value: vm.NewString("table overflow")})
		}
	}

	var tbl *vm.Table
	if narr > 0 || nrec > 0 {
		tbl = vm.NewTableWithSize(int(narr), int(nrec))
	} else {
		tbl = vm.NewEmptyTable()
	}

	v.Set(0, vm.NewTable(tbl))
	return 1
}
