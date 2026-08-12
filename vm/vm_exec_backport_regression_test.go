package vm

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

// Backport of the master-branch vm_exec fixes to the Lua 5.4.8 branch. Two
// groups:
//
//  1. Dispatch correctness: __newindex on a non-table _ENV, call/return hooks
//     for native generic-for iterators and for a body that runs off the end of
//     its code, the frame pointer refetch after __close in the OP_RETURN
//     family, and OP_SELF's string-receiver __index chain.
//
//  2. The OP_CALL "in-loop" fast path: a call to a non-vararg Lua closure with
//     no debug hooks active pushes the callee frame onto vm.callStack and keeps
//     running in the same dispatch loop, instead of recursing into execute()
//     through doCall/vm.call. Every invariant the recursive path used to
//     provide (result adjustment, vm.top discipline, dead register clearing,
//     call-depth accounting, traceback/getlocal visibility, hook events) has to
//     survive that.
//
// Every expectation here was checked against /usr/bin/lua5.4.8.

// backportSetmeta installs a minimal `setmeta(t, mt)` helper so these tests can
// build metatables without pulling in the stdlib (which imports this package).
func backportSetmeta(v *VM) {
	v.SetGlobal("setmeta", NewNativeFunc(func(vm *VM) int {
		obj := vm.Get(1)
		if t := obj.AsTable(); t != nil {
			t.SetMetatable(vm.Get(2).AsTable())
		}
		vm.Set(0, obj)
		return 1
	}))
}

// ---------------------------------------------------------------------------
// Group 1: dispatch correctness
// ---------------------------------------------------------------------------

// OP_SETTABUP raised "attempt to index a X value" whenever _ENV was not a
// table, even though the value carried a __newindex metamethod — while the
// sibling OP_GETTABUP already dispatched __index, so reads through a proxy
// _ENV worked and writes did not. Reference lvm.c routes both through
// luaV_finishget/luaV_finishset.
func TestSetTabUpDispatchesNewIndexOnNonTableEnv(t *testing.T) {
	v := New()
	var gotKey, gotVal Value
	mt := NewEmptyTable()
	mt.SetString(MetaNewIndex, NewNativeFunc(func(vm *VM) int {
		gotKey, gotVal = vm.Get(2), vm.Get(3)
		return 0
	}))
	mt.SetString(MetaIndex, NewNativeFunc(func(vm *VM) int {
		vm.Set(0, NewString("read:"+vm.Get(2).AsString()))
		return 1
	}))
	v.SetGlobal("env", NewUserdataValue(nil, mt))

	// _ENV must be an upvalue of the assigning function for the compiler to
	// emit SETTABUP/GETTABUP rather than a plain register index.
	results, err := runWithVM(t, v, `
local function outer()
	local _ENV = env
	return function() g = 1 return probe end
end
return outer()()`)
	if err != nil {
		t.Fatalf("assignment through a non-table _ENV failed: %v", err)
	}
	if !gotKey.IsString() || gotKey.AsString() != "g" {
		t.Errorf("__newindex key = %v, want \"g\"", gotKey)
	}
	if !gotVal.IsInt() || gotVal.AsInt() != 1 {
		t.Errorf("__newindex value = %v, want 1", gotVal)
	}
	if len(results) != 1 || results[0].AsString() != "read:probe" {
		t.Errorf("GETTABUP through the same _ENV = %v, want \"read:probe\"", results)
	}
}

// A __newindex *table* on a non-table _ENV stores into that table, and a
// chained __newindex resolves the whole way down, exactly as luaV_finishset
// does. Checked against lua5.4.8 via the string metatable.
func TestSetTabUpNewIndexTableAndChainOnNonTableEnv(t *testing.T) {
	v := New()
	backportSetmeta(v)
	sink := NewEmptyTable()
	smeta := NewEmptyTable()
	smeta.SetString(MetaNewIndex, NewTable(sink))
	v.SetStringMeta(smeta)

	if _, err := runWithVM(t, v, `
local function outer()
	local _ENV = "E"
	return function() a = 10 end
end
outer()()`); err != nil {
		t.Fatalf("__newindex table on a string _ENV failed: %v", err)
	}
	if got := sink.GetString("a"); !got.IsInt() || got.AsInt() != 10 {
		t.Errorf("sink.a = %v, want 10", got)
	}

	// Chain: string metatable's __newindex is a table whose own metatable has
	// a __newindex function.
	deep := NewEmptyTable()
	midMeta := NewEmptyTable()
	midMeta.SetString(MetaNewIndex, NewNativeFunc(func(vm *VM) int {
		deep.SetString(vm.Get(2).AsString(), NewInt(vm.Get(3).AsInt()*2))
		return 0
	}))
	mid := NewEmptyTable()
	mid.SetMetatable(midMeta)
	smeta.SetString(MetaNewIndex, NewTable(mid))
	if _, err := runWithVM(t, v, `
local function outer()
	local _ENV = "E"
	return function() c = 21 end
end
outer()()`); err != nil {
		t.Fatalf("chained __newindex on a string _ENV failed: %v", err)
	}
	if !mid.GetString("c").IsNil() {
		t.Errorf("chained __newindex wrote through the intermediate table: %v", mid.GetString("c"))
	}
	if got := deep.GetString("c"); !got.IsInt() || got.AsInt() != 42 {
		t.Errorf("deep.c = %v, want 42", got)
	}
}

// A non-table _ENV without __newindex still has to report the upvalue name,
// and its message must match the read path's.
func TestSetTabUpNonTableEnvErrorKeepsUpvalueName(t *testing.T) {
	for _, src := range []string{
		"local function outer()\n\tlocal _ENV = \"env\"\n\treturn function() g = 1 end\nend\nreturn outer()()",
		"local function outer()\n\tlocal _ENV = \"env\"\n\treturn function() return g end\nend\nreturn outer()()",
	} {
		v := New()
		v.SetStringMeta(NewEmptyTable())
		_, err := runWithVM(t, v, src)
		if err == nil {
			t.Fatalf("%s: expected an error assigning through a string _ENV", src)
		}
		if !strings.Contains(err.Error(), "attempt to index a string value (upvalue '_ENV')") {
			t.Errorf("%s: unexpected error: %v", src, err)
		}
	}
}

// hookEvent is one recorded debug-hook event plus the transfer info the
// returning/called frame reported at the time (what debug.getinfo's ftransfer
// and ntransfer expose).
type hookEvent struct {
	event     string
	ftransfer int
	ntransfer int
	native    bool // the frame the event belongs to is a native function
}

// recordHooks installs a native debug hook that appends one entry per event.
func recordHooks(v *VM, mask byte, log *[]hookEvent) {
	v.SetHook(NewNativeFunc(func(vm *VM) int {
		ev := hookEvent{event: vm.Get(1).AsString()}
		// The hook's own frames sit on top; the event belongs to the first
		// frame below them that is not the hook function itself.
		for i := len(vm.callStack) - 2; i >= 0; i-- {
			f := &vm.callStack[i]
			if f.closure == nil && f.funcValue.IsNativeFunc() && f.callNameWhat == "hook" {
				continue
			}
			ev.ftransfer, ev.ntransfer = f.ftransfer, f.ntransfer
			ev.native = f.closure == nil
			break
		}
		*log = append(*log, ev)
		return 0
	}), mask, 0)
}

// __close runs Lua code from OP_RETURN and can grow (reallocate) the call
// stack; the transfer info for the return hook was then written through the
// stale frame pointer, so the hook saw the frame's initial values instead of
// the returned registers.
func TestReturnHookTransferSurvivesCloseStackGrowth(t *testing.T) {
	v := New()
	backportSetmeta(v)
	var log []hookEvent
	recordHooks(v, HookMaskReturn, &log)

	_, err := runWithVM(t, v, `
local function deep(n)
	if n == 0 then return 0 end
	local r = deep(n - 1)
	return r
end
local mt = setmeta({}, nil)
mt.__close = function() deep(60) end
local function f()
	local x <close> = setmeta({}, mt)
	return 1, 2, 3
end
f()`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	found := false
	for _, e := range log {
		if e.ntransfer == 3 {
			found = true
			if e.ftransfer != 2 {
				t.Errorf("return hook ftransfer = %d, want 2 (lua5.4.8 reports 2/3)", e.ftransfer)
			}
		}
	}
	if !found {
		t.Errorf("no return event reported the 3 returned values; got %+v", log)
	}
}

// A return with no results carries no transfer information: rethook is called
// with ntransfer == 0, luaD_hook never sets CIST_TRAN, and debug.getinfo(...,
// "r") therefore reports 0/0. golua has no CIST_TRAN flag — it reports the
// stored fields verbatim — so the stored pair must be 0/0 for both the
// OP_RETURN0 path and the run-off-the-end path, or debug.getinfo would diverge
// from lua5.4.8. (Master stores the result register instead; that value is
// unobservable there only because of how its debug layer is structured.)
func TestEmptyReturnReportsNoTransfer(t *testing.T) {
	for _, tc := range []struct{ name, src string }{
		{"return0", `local function g() end g()`},
		{"offEnd", `local function g(x) if x then return 1 end end g(false)`},
		{"return0WithLocals", `local function g() local a, b = 1, 2 end g()`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := New()
			var log []hookEvent
			recordHooks(v, HookMaskReturn, &log)
			if _, err := runWithVM(t, v, tc.src); err != nil {
				t.Fatalf("runtime error: %v", err)
			}
			if len(log) == 0 {
				t.Fatal("no return events recorded")
			}
			if log[0].ftransfer != 0 || log[0].ntransfer != 0 {
				t.Errorf("empty return reported %d/%d, want 0/0", log[0].ftransfer, log[0].ntransfer)
			}
		})
	}
}

// luaD_hookcall passes p->numparams as the call hook's ntransfer, so every
// declared parameter slot is transferred — including the ones nil-filled
// because the caller passed fewer arguments. golua counted only the arguments
// actually supplied, so a hook using debug.getlocal over the transfer range
// missed the trailing nil parameters.
func TestCallHookTransfersEveryParameterSlot(t *testing.T) {
	v := New()
	var log []hookEvent
	recordHooks(v, HookMaskCall, &log)
	if _, err := runWithVM(t, v, `
local function two(a, b) return a end
two(1)
two(1, 2)
two(1, 2, 3)`); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	var got []int
	for _, e := range log {
		if !e.native && e.ntransfer != 0 {
			got = append(got, e.ntransfer)
		}
	}
	// Three calls to a two-parameter function, whatever the argument count.
	want := []int{2, 2, 2}
	if len(got) != len(want) {
		t.Fatalf("call ntransfers = %v, want %v (log %+v)", got, want, log)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d ntransfer = %d, want %d", i, got[i], want[i])
		}
	}
}

// The native-iterator branch of OP_TFORCALL invoked the function inline and
// so produced no call/return events: hook-driven counters under-counted
// everything a `for k, v in next, t do` loop did. Reference lvm.c runs every
// iterator through luaD_call. On 5.4 the call is staged at R[A+4], which is
// also where the loop variables live, so no result relocation is needed.
func TestGenericForNativeIteratorFiresCallAndReturnHooks(t *testing.T) {
	v := New()
	// Iterator over a fixed two-element sequence: iter(state, control).
	v.SetGlobal("iter", NewNativeFunc(func(vm *VM) int {
		var i int64
		if c := vm.Get(2); c.IsInt() {
			i = c.AsInt()
		}
		i++
		if i > 2 {
			return 0
		}
		vm.Set(0, NewInt(i))
		vm.Set(1, NewInt(i*10))
		return 2
	}))
	var log []hookEvent
	recordHooks(v, HookMaskCall|HookMaskReturn, &log)

	results, err := runWithVM(t, v, `
local sum = 0
local last
for k, val in iter, {} do sum = sum + val last = k end
return sum, last`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 2 || results[0].AsInt() != 30 || results[1].AsInt() != 2 {
		t.Fatalf("loop results = %v, want [30 2]", results)
	}
	var calls, returns int
	for _, e := range log {
		if !e.native {
			continue // the main chunk's own call/return
		}
		switch e.event {
		case hookEventCall:
			calls++
		case hookEventReturn:
			returns++
		}
	}
	// Two iterations plus the terminating call, each with a matching return.
	if calls != 3 || returns != 3 {
		t.Errorf("native iterator produced %d call / %d return events, want 3 / 3 (log %+v)", calls, returns, log)
	}
}

// The hooked TFORCALL path routes through doCall, which nil-pads the loop
// variables the iterator did not produce. With hooks off the inline path does
// the same; both must agree, and the loop must terminate identically.
func TestGenericForNativeIteratorResultsMatchWithAndWithoutHooks(t *testing.T) {
	src := `
local acc = {}
for a, b, c in iter, {} do acc[#acc+1] = tostring(a) .. "/" .. tostring(b) .. "/" .. tostring(c) end
return table_concat(acc)`
	run := func(hooks bool) string {
		v := New()
		v.SetGlobal("iter", NewNativeFunc(func(vm *VM) int {
			var i int64
			if c := vm.Get(2); c.IsInt() {
				i = c.AsInt()
			}
			i++
			if i > 2 {
				return 0
			}
			vm.Set(0, NewInt(i))
			return 1 // only one result: b and c must be nil-padded
		}))
		v.SetGlobal("tostring", NewNativeFunc(func(vm *VM) int {
			vm.Set(0, NewString(vm.Get(1).String()))
			return 1
		}))
		v.SetGlobal("table_concat", NewNativeFunc(func(vm *VM) int {
			var sb strings.Builder
			tbl := vm.Get(1).AsTable()
			for i := 1; ; i++ {
				el := tbl.Get(NewInt(int64(i)))
				if el.IsNil() {
					break
				}
				if i > 1 {
					sb.WriteString(",")
				}
				sb.WriteString(el.AsString())
			}
			vm.Set(0, NewString(sb.String()))
			return 1
		}))
		if hooks {
			var log []hookEvent
			recordHooks(v, HookMaskCall|HookMaskReturn, &log)
		}
		results, err := runWithVM(t, v, src)
		if err != nil {
			t.Fatalf("hooks=%v: runtime error: %v", hooks, err)
		}
		return results[0].AsString()
	}
	off, on := run(false), run(true)
	if off != "1/nil/nil,2/nil/nil" {
		t.Errorf("hooks off: loop produced %q, want %q", off, "1/nil/nil,2/nil/nil")
	}
	if on != off {
		t.Errorf("hooked iterator path produced %q, unhooked produced %q", on, off)
	}
}

// A Lua iterator whose body ends in an `if` that returns leaves the proto
// without a trailing OP_RETURN0, and running off the end of the code skipped
// the return event entirely.
func TestFallOffEndFiresReturnHook(t *testing.T) {
	v := New()
	var log []hookEvent
	recordHooks(v, HookMaskCall|HookMaskReturn, &log)

	if _, err := runWithVM(t, v, `
local i = 0
local function iter(s, c)
	i = i + 1
	if i <= 2 then return i, i end
end
for k in iter, {} do end`); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	var calls, returns int
	for _, e := range log {
		switch e.event {
		case hookEventCall:
			calls++
		case hookEventReturn:
			returns++
		}
	}
	// Main chunk + three iterator calls, all of them returning.
	if calls != 4 || returns != 4 {
		t.Errorf("got %d call / %d return events, want 4 / 4 (log %+v)", calls, returns, log)
	}
}

// OP_SELF handled only a table or function __index on the string metatable
// and silently produced nil for anything else, so ("s"):foo() and ("s").foo
// disagreed. Both must run the full luaV_finishget chain.
func TestStringMethodIndexChainMatchesFieldAccess(t *testing.T) {
	tests := []struct {
		name  string
		index Value
		want  string
	}{
		{"number __index", NewInt(5), "attempt to index a number value"},
		{"self-referential string __index", NewString("str"), "'__index' chain too long; possible loop"},
		{"missing __index", Nil, "attempt to index a string value"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, src := range []string{`return ("s"):foo()`, `return ("s").foo`} {
				v := New()
				smeta := NewEmptyTable()
				if !tc.index.IsNil() {
					smeta.SetString(MetaIndex, tc.index)
				}
				v.SetStringMeta(smeta)
				_, err := runWithVM(t, v, src)
				if err == nil {
					t.Fatalf("%s: expected an error", src)
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Errorf("%s: error = %v, want it to contain %q", src, err, tc.want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Group 2: the OP_CALL in-loop fast path
// ---------------------------------------------------------------------------

func compileFlat(t *testing.T, source string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse("<flat>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<flat>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

func runFlat(t *testing.T, v *VM, source string) []Value {
	t.Helper()
	results, err := v.Run(compileFlat(t, source))
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return results
}

// installFlatHelpers registers the natives the flattening tests need without
// pulling in the stdlib (which imports this package).
func installFlatHelpers(v *VM) {
	// vals(n) returns n values 1..n — a native multi-return used to build
	// open argument lists (b == 0) for the fast path.
	v.SetGlobal("vals", NewNativeFunc(func(vm *VM) int {
		n := int(vm.Get(1).AsInt())
		for i := 0; i < n; i++ {
			vm.Set(i, NewInt(int64(i+1)))
		}
		return n
	}))
	// pcallv(f, ...) mirrors pcall for the tests: returns ok plus results.
	v.SetGlobal("pcallv", NewNativeFunc(func(vm *VM) int {
		fn := vm.Get(1)
		args := make([]Value, 0, vm.ArgCount()-1)
		for i := 2; i <= vm.ArgCount(); i++ {
			args = append(args, vm.Get(i))
		}
		results, err := vm.ProtectedCall(fn, args)
		if err != nil {
			vm.Set(0, False)
			vm.Set(1, NewString(err.Error()))
			return 2
		}
		vm.Set(0, True)
		for i, r := range results {
			vm.Set(i+1, r)
		}
		return 1 + len(results)
	}))
	v.SetGlobal("setmeta", NewNativeFunc(func(vm *VM) int {
		obj := vm.Get(1)
		if tbl := obj.AsTable(); tbl != nil {
			if mt := vm.Get(2); mt.IsNil() {
				tbl.SetMetatable(nil)
			} else {
				tbl.SetMetatable(mt.AsTable())
			}
		}
		vm.Set(0, obj)
		return 1
	}))
	v.SetGlobal("concat", NewNativeFunc(func(vm *VM) int {
		var sb strings.Builder
		for i := 1; i <= vm.ArgCount(); i++ {
			sb.WriteString(vm.Get(i).String())
		}
		vm.Set(0, NewString(sb.String()))
		return 1
	}))
	// str(v) is tostring, and count(...) is select("#", ...) — the stdlib is
	// not importable from this package.
	v.SetGlobal("str", NewNativeFunc(func(vm *VM) int {
		vm.Set(0, NewString(vm.Get(1).String()))
		return 1
	}))
	v.SetGlobal("count", NewNativeFunc(func(vm *VM) int {
		vm.Set(0, NewInt(int64(vm.ArgCount())))
		return 1
	}))
}

// A flattened call must adjust results exactly like doCall did: truncate to
// C-1 wanted values, nil-pad when the callee produced fewer, and leave vm.top
// marking the count for an open call (C == 0).
func TestFlattenedCallResultAdjustment(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local function none() end
		local function one() return 7 end
		local function three() return 1, 2, 3 end
		local function offEnd(x) if x then return "yes" end end

		local a, b, c = none()
		local d, e = one()
		local f, g, h, i = three()
		local j, k = offEnd(true)
		local l = offEnd(false)
		-- open call: all results forwarded to a native
		local packed = concat(three())
		local trunc = (three())
		return a, b, c, d, e, f, g, h, i, j, k, l, packed, trunc
	`)
	want := []struct {
		nilVal bool
		s      string
	}{
		{nilVal: true}, {nilVal: true}, {nilVal: true}, // none()
		{s: "7"}, {nilVal: true}, // one()
		{s: "1"}, {s: "2"}, {s: "3"}, {nilVal: true}, // three()
		{s: "yes"}, {nilVal: true}, // offEnd(true)
		{nilVal: true}, // offEnd(false) — ran off the end of its code
		{s: "123"},     // open-call arguments
		{s: "1"},       // parenthesized truncation
	}
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d: %v", len(results), len(want), results)
	}
	for i, w := range want {
		if w.nilVal {
			if !results[i].IsNil() {
				t.Errorf("result %d = %v, want nil", i, results[i].String())
			}
			continue
		}
		if got := results[i].String(); got != w.s {
			t.Errorf("result %d = %q, want %q", i, got, w.s)
		}
	}
}

// An open call (B == 0) can leave its arguments above the caller's frame top,
// so the argument copy into the callee's registers overlaps the source range.
// It must have memmove semantics, and the callee must see the right values.
func TestFlattenedCallOpenArgumentOverlap(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local function three(a, b, c)
			return str(a) .. "/" .. str(b) .. "/" .. str(c)
		end
		local function two(a, b) return (a or 0) * 100 + (b or 0) end
		local function lots(a1,a2,a3,a4,a5,a6,a7,a8,a9,a10)
			return (a1 or 0)+(a2 or 0)+(a3 or 0)+(a4 or 0)+(a5 or 0)
				+(a6 or 0)+(a7 or 0)+(a8 or 0)+(a9 or 0)+(a10 or 0)
		end
		local function multi() return 4, 5, 6 end
		return three(vals(3)), three(vals(1)), three(vals(0)),
			two(vals(40)), lots(vals(10)), three(multi()), two(0, multi())
	`)
	got := make([]string, len(results))
	for i, r := range results {
		got[i] = r.String()
	}
	want := []string{"1/2/3", "1/nil/nil", "nil/nil/nil", "102", "55", "4/5/6", "4"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// The two dead-register clear loops on the recursive path (one in doCall, one
// in vm.call) are a GC-liveness contract: without them a value that is only
// reachable from a dead register keeps its object alive. popInLoopFrame must
// clear the union of both ranges — everything above the caller's result area
// through the end of the callee's register window.
func TestFlattenedCallClearsDeadRegisters(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	marker := NewEmptyTable()
	markerVal := NewTable(marker)
	v.SetGlobal("marker", markerVal)

	var leaked int
	var scanned bool
	v.SetGlobal("probe", NewNativeFunc(func(vm *VM) int {
		// callStack: [... driver, probe]. Everything at or above the driver's
		// frame top belongs to the callee window that just returned.
		if len(vm.callStack) < 2 {
			return 0
		}
		driver := &vm.callStack[len(vm.callStack)-2]
		if driver.closure == nil {
			return 0
		}
		scanned = true
		driverTop := driver.base + driver.closure.Proto.MaxStack
		for i := driverTop; i < len(vm.stack); i++ {
			if vm.stack[i].AsTable() == marker {
				leaked++
			}
		}
		return 0
	}))

	runFlat(t, v, `
		local function callee(t)
			local a, b, c = t, t, t
			local d = {a, b, c}
			return 1
		end
		local function driver()
			local r = callee(marker)
			probe()
			return r
		end
		return driver()
	`)
	if !scanned {
		t.Fatal("probe never ran with a Lua caller frame")
	}
	if leaked != 0 {
		t.Errorf("marker table still referenced from %d dead register(s) above the caller frame", leaked)
	}
}

// flattenWideLocals returns n statements binding distinct locals to m, so the
// enclosing function's register window is at least n slots wide and every one
// of those registers holds whatever m is.
func flattenWideLocals(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "\t\t\t\tlocal w%d = m\n", i)
	}
	return sb.String()
}

// flattenLeakScan runs source in a fresh VM with `marker` (a table) and
// `probe` (a native) installed, and reports how many stack slots at or above
// probe's Lua caller's frame top still reference the marker table. Anything
// above that frame top is a dead register of a call that already returned.
func flattenLeakScan(t *testing.T, source string) ([]Value, int) {
	t.Helper()
	v := New()
	installFlatHelpers(v)
	marker := NewEmptyTable()
	v.SetGlobal("marker", NewTable(marker))

	leaked := -1
	v.SetGlobal("probe", NewNativeFunc(func(vm *VM) int {
		if len(vm.callStack) < 2 {
			return 0
		}
		caller := &vm.callStack[len(vm.callStack)-2]
		if caller.closure == nil {
			return 0
		}
		leaked = 0
		callerTop := caller.base + caller.closure.Proto.MaxStack
		for i := callerTop; i < len(vm.stack); i++ {
			if vm.stack[i].AsTable() == marker {
				leaked++
			}
		}
		return 0
	}))

	results := runFlat(t, v, source)
	if leaked < 0 {
		t.Fatal("probe never ran with a Lua caller frame")
	}
	return results, leaked
}

// OP_TAILCALL reuses the frame in place and swaps frame.closure, so a
// flattened callee can finish running under a proto whose MaxStack is smaller
// than the window that was pushed for it. The registers between the two tops
// are dead but were never overwritten, and the return-time clear in
// popInLoopFrame is bounded by the *current* proto, so they would keep their
// objects alive. The recursive path had no such hole: vm.call captures the
// entry proto and always clears the full original window. The shrink has to
// be cleared as it happens.
func TestFlattenedCallTailCallShrinksRegisterWindow(t *testing.T) {
	// One shrink: a wide frame tail-calls a narrow one.
	results, leaked := flattenLeakScan(t, `
		local function small(x) return x end
		local function big(m)
`+flattenWideLocals(40)+`
			return small(1)
		end
		local function driver()
			local v = big(marker)
			probe()
			return v
		end
		return driver()
	`)
	if len(results) != 1 || results[0].String() != "1" {
		t.Fatalf("tail call through a shrinking frame returned %v, want [1]", results)
	}
	if leaked != 0 {
		t.Errorf("single shrink: marker still referenced from %d dead register(s) above the caller frame", leaked)
	}

	// Chained shrinks: each tail call abandons another slice of the window,
	// so the clears have to compose.
	_, leaked = flattenLeakScan(t, `
		local function tiny(x) return x end
		local function mid(m)
`+flattenWideLocals(12)+`
			return tiny(1)
		end
		local function big(m)
`+flattenWideLocals(40)+`
			return mid(m)
		end
		local function driver()
			local v = big(marker)
			probe()
			return v
		end
		return driver()
	`)
	if leaked != 0 {
		t.Errorf("chained shrink: marker still referenced from %d dead register(s) above the caller frame", leaked)
	}

	// A shrink followed by a native tail call: the flattened frame's results
	// come from the native, and popInLoopFrame still has to clear the whole
	// window that was pushed for it.
	_, leaked = flattenLeakScan(t, `
		local function narrow(m) return vals(2) end
		local function big(m)
`+flattenWideLocals(40)+`
			return narrow(m)
		end
		local function driver()
			local a, b = big(marker)
			probe()
			return a, b
		end
		return driver()
	`)
	if leaked != 0 {
		t.Errorf("shrink then native tail call: marker still referenced from %d dead register(s) above the caller frame", leaked)
	}
}

// The other direction: the tail-called proto has a LARGER MaxStack than the
// window pushed for the flattened frame, so the dead window at return time is
// wider than what was pushed. Bounding the clear by the pushed window (or by
// vm.top, which a nested call lowers) would leave that growth behind.
func TestFlattenedCallTailCallGrowsRegisterWindow(t *testing.T) {
	results, leaked := flattenLeakScan(t, `
		local function wide(m)
`+flattenWideLocals(40)+`
			return 1
		end
		local function narrow(m)
			return wide(m)
		end
		local function driver()
			local v = narrow(marker)
			probe()
			return v
		end
		return driver()
	`)
	if len(results) != 1 || results[0].String() != "1" {
		t.Fatalf("tail call through a growing frame returned %v, want [1]", results)
	}
	if leaked != 0 {
		t.Errorf("marker still referenced from %d dead register(s) above the caller frame", leaked)
	}

	// Grow after a nested flattened call has lowered vm.top: the nested call
	// returns through popInLoopFrame, which parks vm.top at the *current*
	// frame's top, well below the registers the grown proto goes on to use.
	_, leaked = flattenLeakScan(t, `
		local function nested(x) return x end
		local function wide(m)
			local seed = nested(m)
`+flattenWideLocals(40)+`
			return 1
		end
		local function narrow(m)
			return wide(m)
		end
		local function driver()
			local v = narrow(marker)
			probe()
			return v
		end
		return driver()
	`)
	if leaked != 0 {
		t.Errorf("grow after a nested call: marker still referenced from %d dead register(s) above the caller frame", leaked)
	}
}

// Depth accounting stays on vm.callStack, so a runaway flattened recursion
// must still overflow at the same depth, produce "stack overflow", be
// catchable, and leave the VM usable afterwards.
func TestFlattenedCallStackOverflowCatchable(t *testing.T) {
	v := New(WithLimits(Limits{MaxCallDepth: 400}))
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local function rec(n) return rec(n + 1) + 1 end
		local ok, err = pcallv(rec, 1)
		-- the VM must still work after the overflow was caught
		local function add(a, b) return a + b end
		local s = 0
		for i = 1, 100 do s = add(s, i) end
		return ok, err, s
	`)
	if len(results) != 3 {
		t.Fatalf("got %d results: %v", len(results), results)
	}
	if results[0].ToBool() {
		t.Fatalf("expected the recursion to overflow, got ok=true")
	}
	if msg := results[1].String(); !strings.Contains(msg, "stack overflow") {
		t.Errorf("error = %q, want it to mention stack overflow", msg)
	}
	if results[2].String() != "5050" {
		t.Errorf("post-overflow arithmetic = %v, want 5050", results[2].String())
	}
}

// The exact depth at which pure Lua recursion overflows must not move: it is
// vm.callStack length that is limited, and the fast path pushes exactly one
// frame per call just as vm.call did (checkCallDepth runs before the push in
// both). The control is the same recursion through a vararg callee, which the
// fast path declines, so both counts come out of the same VM configuration.
func TestFlattenedCallOverflowDepthUnchanged(t *testing.T) {
	depthOf := func(src string) int64 {
		v := New(WithLimits(Limits{MaxCallDepth: 250}))
		installFlatHelpers(v)
		results := runFlat(t, v, src)
		if results[0].ToBool() {
			t.Fatal("expected an overflow")
		}
		return results[1].AsInt()
	}
	flat := depthOf(`
		local d = 0
		local function rec() d = d + 1 return rec() + 1 end
		local ok = pcallv(rec)
		return ok, d
	`)
	// Vararg callees never take the fast path: this recursion still goes
	// through doCall -> vm.call -> execute.
	recursive := depthOf(`
		local d = 0
		local function rec(...) d = d + 1 return rec() + 1 end
		local ok = pcallv(rec)
		return ok, d
	`)
	if flat != recursive {
		t.Errorf("flattened recursion overflowed at depth %d, the recursive path at %d; the limit must not move", flat, recursive)
	}
	// main chunk + pcallv native + the recursive frames = MaxCallDepth.
	if flat != 249 {
		t.Errorf("reached depth %d before overflow, want 249 (MaxCallDepth 250 minus the main chunk frame)", flat)
	}
}

// The flattened frames must be real frames: tracebacks, StackDepth and
// GetLocal all walk vm.callStack, so a native called at the bottom of a chain
// of flattened calls has to see every level.
func TestFlattenedCallFramesVisibleToDebug(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	var depth int
	var tb string
	var locals []string
	v.SetGlobal("inspect", NewNativeFunc(func(vm *VM) int {
		depth = vm.StackDepth()
		tb = vm.Traceback("msg", 1)
		for lvl := 1; lvl <= 3; lvl++ {
			name, val, ok := vm.GetLocal(lvl, 1)
			if !ok {
				locals = append(locals, "-")
				continue
			}
			locals = append(locals, name+"="+val.String())
		}
		return 0
	}))
	runFlat(t, v, `
		-- deliberately NOT tail calls: each frame must stay on the stack
		local function m3(c) local zz = c local r = inspect() return r end
		local function m2(b) local yy = b + 1 local r = m3(yy) return r end
		local function m1(a) local xx = a + 1 local r = m2(xx) return r end
		local top = m1(1)
		return top
	`)
	// main chunk + m1 + m2 + m3 + inspect = 5 frames.
	if depth != 5 {
		t.Errorf("StackDepth() = %d, want 5 (main, m1, m2, m3, inspect)", depth)
	}
	if n := strings.Count(tb, "<flat>"); n < 4 {
		t.Errorf("traceback lists %d <flat> frames, want at least 4:\n%s", n, tb)
	}
	want := []string{"c=3", "b=2", "a=1"}
	if len(locals) != len(want) {
		t.Fatalf("locals = %v, want %v", locals, want)
	}
	for i := range want {
		if locals[i] != want[i] {
			t.Errorf("locals[%d] = %q, want %q", i, locals[i], want[i])
		}
	}
}

// An error raised inside a flattened frame must carry the same position
// information and be catchable by ProtectedCall, which truncates the call
// stack back to its own depth.
func TestFlattenedCallErrorPositionAndRecovery(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local function bad()
			local t = nil
			return t.field
		end
		local function mid() local r = bad() return r end
		local ok, err = pcallv(mid)
		return ok, err
	`)
	if results[0].ToBool() {
		t.Fatalf("expected an error, got ok=true")
	}
	msg := results[1].String()
	if !strings.Contains(msg, ":4:") || !strings.Contains(msg, "attempt to index a nil value") {
		t.Errorf("error = %q, want it positioned at <flat>:4 with an index error", msg)
	}
	if len(v.callStack) != 0 {
		t.Errorf("call stack not unwound after the caught error: %d frames", len(v.callStack))
	}
}

// A to-be-closed variable declared inside a flattened callee must run its
// __close handler when that callee returns, before the caller resumes.
func TestFlattenedCallClosesTBCVariables(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	var order []string
	v.SetGlobal("note", NewNativeFunc(func(vm *VM) int {
		order = append(order, vm.Get(1).String())
		return 0
	}))
	results := runFlat(t, v, `
		local mt = {}
		mt.__close = function(self, err) note("close:" .. self.name) end
		local function callee(n)
			local guard <close> = setmeta({name = "c" .. n}, mt)
			note("enter:" .. n)
			if n > 0 then return callee(n - 1) + 1 end
			return 0
		end
		local function driver()
			local r = callee(2)
			note("after")
			return r
		end
		return driver()
	`)
	if got := results[0].String(); got != "2" {
		t.Errorf("result = %q, want 2", got)
	}
	want := "enter:2 enter:1 enter:0 close:c0 close:c1 close:c2 after"
	if got := strings.Join(order, " "); got != want {
		t.Errorf("event order = %q, want %q", got, want)
	}
}

// A tail call to a native from a flattened frame produces that frame's
// results: they have to be adjusted into the *caller's* registers, since
// there is no vm.call on the Go stack to hand them to.
func TestFlattenedCallNativeTailCall(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local function tailToNative(n) return vals(n) end
		local function driver()
			local a, b, c = tailToNative(3)
			local only = tailToNative(2)
			local none = tailToNative(0)
			return a, b, c, only, none, tailToNative(4)
		end
		return driver()
	`)
	got := make([]string, len(results))
	for i, r := range results {
		got[i] = r.String()
	}
	want := []string{"1", "2", "3", "1", "nil", "1", "2", "3", "4"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// With hooks active the fast path is disabled entirely, so every call and
// return event must still fire. A hook enabled *during* a flattened callee
// must also see that callee's return event.
func TestFlattenedCallHookEvents(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	var events []string
	hook := NewNativeFunc(func(vm *VM) int {
		events = append(events, vm.Get(1).String())
		return 0
	})
	v.SetGlobal("hookon", NewNativeFunc(func(vm *VM) int {
		vm.SetHook(hook, HookMaskCall|HookMaskReturn, 0)
		return 0
	}))
	v.SetGlobal("hookoff", NewNativeFunc(func(vm *VM) int {
		vm.SetHook(Nil, 0, 0)
		return 0
	}))

	// Hooks active before the calls: doCall handles everything.
	runFlat(t, v, `
		local function leaf() return 1 end
		local function mid() return leaf() + 1 end
		hookon()
		local r = mid()
		hookoff()
		return r
	`)
	calls, rets := 0, 0
	for _, e := range events {
		switch e {
		case "call":
			calls++
		case "return":
			rets++
		}
	}
	if calls < 2 || rets < 2 {
		t.Errorf("hooks active up front: got %d call / %d return events (%v), want at least 2 each", calls, rets, events)
	}

	// Hook enabled mid-flight, inside a frame the fast path already pushed:
	// that frame's return event must still fire.
	events = nil
	v2 := New()
	installFlatHelpers(v2)
	v2.SetGlobal("hookon", NewNativeFunc(func(vm *VM) int {
		vm.SetHook(hook, HookMaskCall|HookMaskReturn, 0)
		return 0
	}))
	v2.SetGlobal("hookoff", NewNativeFunc(func(vm *VM) int {
		vm.SetHook(Nil, 0, 0)
		return 0
	}))
	runFlat(t, v2, `
		local function inner(x)
			hookon()
			return x + 1
		end
		local function outer(x)
			local y = inner(x)
			return y * 2
		end
		local r = outer(10)
		hookoff()
		return r
	`)
	rets = 0
	for _, e := range events {
		if e == "return" {
			rets++
		}
	}
	if rets < 2 {
		t.Errorf("mid-call sethook: got %d return events (%v), want at least 2 (inner and outer)", rets, events)
	}
}

// vm.top has to be exact after a flattened call, including for an open call
// that lowers it below the caller's frame top. A stale vm.top previously let
// metamethod dispatch scribble over the caller's registers.
func TestFlattenedCallTopDisciplineAcrossMetamethods(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local mt = {}
		mt.__index = function(self, k) return "idx:" .. k end
		mt.__add = function(a, b) return 99 end
		local function multi() return 1, 2, 3 end
		local function driver()
			local keep1, keep2 = "K1", "K2"
			-- open call lowers vm.top to mark the result count
			local packed = concat(multi())
			local obj = setmeta({}, mt)
			-- metamethod dispatch immediately afterwards must not corrupt
			-- the caller's live registers
			local viaIndex = obj.field
			local viaAdd = obj + 1
			return keep1, keep2, packed, viaIndex, viaAdd
		end
		return driver()
	`)
	got := make([]string, len(results))
	for i, r := range results {
		got[i] = r.String()
	}
	want := []string{"K1", "K2", "123", "idx:field", "99"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result %d = %q, want %q", i, got[i], want[i])
		}
	}
	if v.top != 0 && len(v.callStack) != 0 {
		t.Errorf("vm.top = %d with %d frames left after Run", v.top, len(v.callStack))
	}
}

// Vararg callees, __call objects and natives are excluded from the fast path
// and must keep behaving exactly as before, including when they are mixed
// into a chain of flattened calls.
func TestFlattenedCallExcludedCalleesUnaffected(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local function va(...)
			local n = count(...)
			return n, ...
		end
		local function useVa() return va(1, nil, 3) end
		local callable = setmeta({}, {__call = function(self, a, b) return a + b end})
		local function useCallable() return callable(3, 4) end
		local function useNative() return concat("a", "b") end
		local function mixed()
			local n, x, y, z = useVa()
			return n, x, y, z, useCallable(), useNative()
		end
		return mixed()
	`)
	got := make([]string, len(results))
	for i, r := range results {
		got[i] = r.String()
	}
	want := []string{"3", "1", "nil", "3", "7", "ab"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// A closure created inside a flattened callee must have its upvalues closed
// when that callee returns, so each iteration keeps its own value.
func TestFlattenedCallClosesUpvaluesOnReturn(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	results := runFlat(t, v, `
		local fns = {}
		local function maker(i)
			local x = i * 10
			fns[i] = function() return x end
			return i
		end
		local function driver()
			for i = 1, 3 do maker(i) end
			return fns[1]() + fns[2]() + fns[3]()
		end
		return driver()
	`)
	if got := results[0].String(); got != "60" {
		t.Errorf("sum of captured upvalues = %q, want 60", got)
	}
}

// A coroutine body is executed by its own VM, so its dispatch loop has its own
// entryDepth. Yielding out of and resuming back into a chain of flattened
// frames must preserve every level's registers.
func TestFlattenedCallAcrossCoroutineYield(t *testing.T) {
	parent := New()
	installFlatHelpers(parent)
	co := NewCoroutineVM(parent, make(chan []Value), make(chan []Value), 1)
	installFlatHelpers(co)

	proto := compileFlat(t, `
		local function inner(x)
			local keep = x * 2
			local got = yielder(x + 1)
			return got * 10 + keep
		end
		local function mid(x)
			local seen = x + 100
			return inner(x) + seen
		end
		local out = mid(1)
		return out
	`)

	// yielder() stands in for coroutine.yield: it records the value and hands
	// back a fixed reply, exercising a native call at the bottom of a chain of
	// flattened frames (a real yield parks the same Go goroutine there).
	var yielded []string
	co.SetGlobal("yielder", NewNativeFunc(func(vm *VM) int {
		yielded = append(yielded, vm.Get(1).String())
		// Walk the frames the way the resume path does, to be sure they are
		// all present and intact while the native is on top.
		if vm.StackDepth() != 4 {
			t.Errorf("StackDepth inside the yield stand-in = %d, want 4", vm.StackDepth())
		}
		vm.Set(0, NewInt(5))
		return 1
	}))
	results, err := co.Run(proto)
	if err != nil {
		t.Fatalf("coroutine body failed: %v", err)
	}
	if len(yielded) != 1 || yielded[0] != "2" {
		t.Errorf("yielded %v, want [2]", yielded)
	}
	// inner: 5*10 + 1*2 = 52; mid: 52 + 101 = 153.
	if got := results[0].String(); got != "153" {
		t.Errorf("result = %q, want 153", got)
	}
}

// The point of the fast path: nested Lua-to-Lua calls no longer consume Go
// stack. Measured from a native at the bottom of the chain, the number of Go
// frames must not grow with the Lua call depth. This is also what makes deep
// recursion cheap, and it is the structural property the optimization is
// claimed on, so guard it directly.
func TestFlattenedCallDoesNotGrowGoStack(t *testing.T) {
	v := New()
	installFlatHelpers(v)
	var goFrames []int
	v.SetGlobal("goframes", NewNativeFunc(func(vm *VM) int {
		pcs := make([]uintptr, 4096)
		goFrames = append(goFrames, runtime.Callers(0, pcs))
		return 0
	}))
	runFlat(t, v, `
		local function chain(n)
			if n == 0 then
				goframes()
				return 0
			end
			local r = chain(n - 1)
			return r + 1
		end
		local a = chain(2)
		local b = chain(60)
		return a, b
	`)
	if len(goFrames) != 2 {
		t.Fatalf("goframes called %d times, want 2", len(goFrames))
	}
	if goFrames[1] != goFrames[0] {
		t.Errorf("Go stack grew with Lua call depth: %d frames at Lua depth 2, %d at depth 60 "+
			"(the OP_CALL fast path should keep this constant)", goFrames[0], goFrames[1])
	}
}

// A benchmark whose only variable is the number of Lua-to-Lua calls: the
// per-call cost is what the fast path changes. Timing is deliberately not
// asserted here (this project's sec/op measurements are only meaningful on a
// quiet host); the benchmark exists to show the path stays allocation-free.
func BenchmarkFlattenedCallDepth(b *testing.B) {
	source := `
		local function leaf(x) return x + 1 end
		local function l1(x) return leaf(x) end
		local function l2(x) return l1(x) end
		local function l3(x) return l2(x) end
		local function l4(x) return l3(x) end
		local s = 0
		for i = 1, 200 do s = l4(s) end
		return s
	`
	block, err := parser.Parse("<bench>", source)
	if err != nil {
		b.Fatal(err)
	}
	proto, err := compiler.Compile("<bench>", block)
	if err != nil {
		b.Fatal(err)
	}
	v := New()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := v.Run(proto); err != nil {
			b.Fatal(err)
		}
	}
}
