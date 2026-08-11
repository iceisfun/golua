package vm

import (
	"strings"
	"testing"
)

// setmetaGlobal installs a minimal `setmeta(t, mt)` helper so these tests can
// build metatables without pulling in the stdlib (which imports this package).
func setmetaGlobal(v *VM) {
	v.SetGlobal("setmeta", NewNativeFunc(func(vm *VM) int {
		obj := vm.Get(1)
		if t := obj.AsTable(); t != nil {
			t.SetMetatable(vm.Get(2).AsTable())
		}
		vm.Set(0, obj)
		return 1
	}))
}

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

// A non-table _ENV without __newindex still has to report the upvalue name.
func TestSetTabUpNonTableEnvErrorKeepsUpvalueName(t *testing.T) {
	v := New()
	_, err := runWithVM(t, v, `
local function outer()
	local _ENV = "env"
	return function() g = 1 end
end
return outer()()`)
	if err == nil {
		t.Fatal("expected an error assigning through a string _ENV")
	}
	if !strings.Contains(err.Error(), "attempt to index a string value (upvalue '_ENV')") {
		t.Errorf("unexpected error: %v", err)
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
	setmetaGlobal(v)
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
				t.Errorf("return hook ftransfer = %d, want 2 (reference reports 2/3)", e.ftransfer)
			}
		}
	}
	if !found {
		t.Errorf("no return event reported the 3 returned values; got %+v", log)
	}
}

// OP_RETURN0 hardcoded ftransfer 0; reference's rethook reports the register
// the (empty) result list starts at, i.e. the instruction's A operand + 1.
func TestReturn0ReportsResultRegister(t *testing.T) {
	v := New()
	var log []hookEvent
	recordHooks(v, HookMaskReturn, &log)

	if _, err := runWithVM(t, v, `local function g() end g()`); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(log) == 0 {
		t.Fatal("no return events recorded")
	}
	first := log[0]
	if first.ntransfer != 0 || first.ftransfer != 1 {
		t.Errorf("empty return reported %d/%d, want 1/0", first.ftransfer, first.ntransfer)
	}
}

// The native-iterator branch of OP_TFORCALL invoked the function inline and
// so produced no call/return events: hook-driven counters under-counted
// everything a `for k, v in next, t do` loop did. Reference lvm.c runs every
// iterator through luaD_call.
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
for k, val in iter, {} do sum = sum + val end
return sum`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 || results[0].AsInt() != 30 {
		t.Fatalf("loop body result = %v, want 30", results)
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
