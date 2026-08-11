package vm

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

// These tests cover the OP_CALL "in-loop" fast path: a call to a non-vararg
// Lua closure with no debug hooks active pushes the callee frame onto
// vm.callStack and keeps running in the same dispatch loop, instead of
// recursing into execute() through doCall/vm.call. Every invariant the
// recursive path used to provide (result adjustment, vm.top discipline, dead
// register clearing, call-depth accounting, traceback/getlocal visibility,
// hook events) has to survive that.

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
	if n := strings.Count(tb, `"<flat>"]:`); n < 4 {
		t.Errorf("traceback lists %d Lua frames, want at least 4:\n%s", n, tb)
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
	if !strings.Contains(msg, `"<flat>"]:4:`) || !strings.Contains(msg, "attempt to index a nil value") {
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

// The point of the fast path: nested Lua-to-Lua calls no longer consume Go
// stack. Measured from a native at the bottom of the chain, the number of Go
// frames must not grow with the Lua call depth. This is also what makes the
// test above (deep recursion) cheap, and it is the structural property the
// optimization is claimed on, so guard it directly.
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
