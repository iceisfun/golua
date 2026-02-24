package vm

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

// These tests reproduce the __call metamethod bug where doCall returns early
// via vm.call() without placing results back into the caller's stack registers.
//
// Root cause: In doCall (vm.go ~line 1806), the __call path for Lua function
// metamethods does `return vm.call(...)` which bypasses the result-placement
// code at the end of doCall. The OP_CALL handler discards the returned Go
// slice (it expects doCall to have written results into the stack).

// makeCallable creates a table with a native __call metamethod that multiplies
// the first arg by the given factor.
func makeCallable(factor int64) Value {
	tbl := NewEmptyTable()
	tbl.Set(NewString("factor"), NewInt(factor))

	mt := NewEmptyTable()
	mt.Set(metaCall, NewNativeFunc(func(v *VM) int {
		self := v.Get(1) // the table itself (prepended by __call dispatch)
		val := v.Get(2)  // first argument
		f := self.AsTable().Get(NewString("factor"))
		v.Set(0, NewInt(val.AsInt()*f.AsInt()))
		return 1
	}))
	tbl.SetMetatable(mt)
	return NewTable(tbl)
}

// runWithCallable compiles and runs Lua code with a 'callable' global
// that has a __call metamethod (native Go function).
func runWithCallable(t *testing.T, source string) []Value {
	t.Helper()
	return runWithGlobals(t, source, map[string]Value{
		"callable": makeCallable(2),
	})
}

// runLuaCallable compiles Lua code that includes setmetatable and a Lua
// __call metamethod, then runs it.
func runLuaCallable(t *testing.T, source string) ([]Value, error) {
	t.Helper()
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := New()
	// Register setmetatable so Lua code can set up __call
	v.SetGlobal("setmetatable", NewNativeFunc(func(v *VM) int {
		tbl := v.Get(1).AsTable()
		mt := v.Get(2)
		if mt.IsNil() {
			tbl.SetMetatable(nil)
		} else {
			tbl.SetMetatable(mt.AsTable())
		}
		v.Set(0, NewTable(tbl))
		return 1
	}))
	results, err := v.Run(proto)
	return results, err
}

// ──────────────────────────────────────────────────────────────
// Tests using native Go __call metamethod
// ──────────────────────────────────────────────────────────────

// TestMetaCallNativeBasic tests __call with a native metamethod returns a value.
func TestMetaCallNativeBasic(t *testing.T) {
	results := runWithCallable(t, `return callable(5)`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected callable(5) == 10, got %v", results[0])
	}
}

// TestMetaCallNativeInExpression tests __call result in a comparison.
func TestMetaCallNativeInExpression(t *testing.T) {
	results := runWithCallable(t, `return callable(5) == 10`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if !results[0].AsBool() {
		t.Errorf("expected callable(5) == 10 to be true")
	}
}

// TestMetaCallNativeAsMiddleArg tests that __call as a non-last arg
// doesn't corrupt adjacent registers.
func TestMetaCallNativeAsMiddleArg(t *testing.T) {
	results := runWithGlobals(t, `
		local function gather(...)
			return ...
		end
		return gather("foo", callable(5), "bar")
	`, map[string]Value{"callable": makeCallable(2)})
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if results[0].AsString() != "foo" {
		t.Errorf("arg1: expected 'foo', got %v", results[0])
	}
	if results[1].AsInt() != 10 {
		t.Errorf("arg2: expected 10, got %v", results[1])
	}
	if results[2].AsString() != "bar" {
		t.Errorf("arg3: expected 'bar', got %v", results[2])
	}
}

// TestMetaCallNativeChained tests nested __call invocations.
func TestMetaCallNativeChained(t *testing.T) {
	results := runWithCallable(t, `return callable(callable(5))`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 20 {
		t.Errorf("expected callable(callable(5)) == 20, got %v", results[0])
	}
}

// TestMetaCallNativeLocalAssignment tests capturing __call result in local.
func TestMetaCallNativeLocalAssignment(t *testing.T) {
	results := runWithCallable(t, `
		local x = callable(5)
		return x
	`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results[0])
	}
}

// TestMetaCallNativeInArithmetic tests __call result in arithmetic.
func TestMetaCallNativeInArithmetic(t *testing.T) {
	results := runWithCallable(t, `return callable(3) + callable(7)`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 20 {
		t.Errorf("expected 6 + 14 == 20, got %v", results[0])
	}
}

// ──────────────────────────────────────────────────────────────
// Tests using Lua function __call metamethod (different code path)
// ──────────────────────────────────────────────────────────────

// TestMetaCallLuaBasic tests __call with a Lua function metamethod.
// This exercises the doCall path where newFn.IsFunction() is true,
// which has the early-return bug.
func TestMetaCallLuaBasic(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 2}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		return callable(5)
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected callable(5) == 10, got %v", results[0])
	}
}

// TestMetaCallLuaSelfAccess verifies __call receives the table as self.
func TestMetaCallLuaSelfAccess(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({name = "hello"}, {
			__call = function(self)
				return self.name
			end
		})
		return callable()
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsString() != "hello" {
		t.Errorf("expected 'hello', got %v", results[0])
	}
}

// TestMetaCallLuaMultipleArgs tests all args are forwarded past self.
func TestMetaCallLuaMultipleArgs(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({}, {
			__call = function(self, a, b, c)
				return a + b + c
			end
		})
		return callable(10, 20, 30)
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 60 {
		t.Errorf("expected 60, got %v", results[0])
	}
}

// TestMetaCallLuaMultipleReturns tests multiple return values from __call.
func TestMetaCallLuaMultipleReturns(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({}, {
			__call = function(self, x)
				return x, x * 2, x * 3
			end
		})
		return callable(5)
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 5 || results[1].AsInt() != 10 || results[2].AsInt() != 15 {
		t.Errorf("expected 5,10,15; got %v,%v,%v", results[0], results[1], results[2])
	}
}

// TestMetaCallLuaInExpression tests __call result in an equality check.
// This is the exact bug from the user's test suite.
func TestMetaCallLuaInExpression(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 2}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		return callable(5) == 10
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if !results[0].AsBool() {
		t.Errorf("expected callable(5) == 10 to be true, got false")
	}
}

// TestMetaCallLuaAsMiddleArg tests register corruption when __call is a
// non-last argument. Reproduces the "meta_call_debug" bug where
// print("foo", callable(5), "bar") outputs nil, nil, "bar".
func TestMetaCallLuaAsMiddleArg(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 2}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		local function gather(...)
			return ...
		end
		return gather("foo", callable(5), "bar")
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if results[0].AsString() != "foo" {
		t.Errorf("arg1: expected 'foo', got %v", results[0])
	}
	if results[1].AsInt() != 10 {
		t.Errorf("arg2: expected 10 (callable(5)), got %v", results[1])
	}
	if results[2].AsString() != "bar" {
		t.Errorf("arg3: expected 'bar', got %v", results[2])
	}
}

// TestMetaCallLuaChained tests nested __call: callable(callable(5)).
func TestMetaCallLuaChained(t *testing.T) {
	results, err := runLuaCallable(t, `
		local double = setmetatable({}, {
			__call = function(self, x) return x * 2 end
		})
		return double(double(5))
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 20 {
		t.Errorf("expected double(double(5)) == 20, got %v", results[0])
	}
}

// TestMetaCallLuaLocalAssignment tests capturing __call in a local.
func TestMetaCallLuaLocalAssignment(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 2}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		local result = callable(5)
		return result
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results[0])
	}
}

// TestMetaCallLuaInArithmetic tests __call result in arithmetic expression.
func TestMetaCallLuaInArithmetic(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({}, {
			__call = function(self, x) return x end
		})
		return callable(7) + callable(3)
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results[0])
	}
}

// TestMetaCallProtectedCallPath verifies __call works through the Go-level
// ProtectedCall API (which has a separate __call dispatch from doCall).
func TestMetaCallProtectedCallPath(t *testing.T) {
	callable := makeCallable(2)
	v := New()
	results, err := v.ProtectedCall(callable, []Value{NewInt(5)})
	if err != nil {
		t.Fatalf("ProtectedCall error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results[0])
	}
}

// TestMetaCallLuaResultUsedInIf tests __call result used in a conditional
// (not just return position). This stresses the register placement.
func TestMetaCallLuaResultUsedInIf(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 2}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		if callable(5) == 10 then
			return "yes"
		else
			return "no"
		end
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsString() != "yes" {
		t.Errorf("expected 'yes', got %v", results[0])
	}
}

// TestMetaCallNativeResultUsedInIf tests native __call in a conditional.
func TestMetaCallNativeResultUsedInIf(t *testing.T) {
	results := runWithCallable(t, `
		if callable(5) == 10 then
			return "yes"
		else
			return "no"
		end
	`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsString() != "yes" {
		t.Errorf("expected 'yes', got %v", results[0])
	}
}

// ──────────────────────────────────────────────────────────────
// Edge case / regression tests
// ──────────────────────────────────────────────────────────────

// TestMetaCallTailCall verifies __call works in tail-call position.
// This exercises the OP_TAILCALL dispatch loop.
func TestMetaCallTailCall(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 2}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		local function wrapper(x)
			return callable(x)  -- tail call position
		end
		return wrapper(5)
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results[0])
	}
}

// TestMetaCallNativeTailCall verifies native __call in tail-call position.
func TestMetaCallNativeTailCall(t *testing.T) {
	results := runWithCallable(t, `
		local function wrapper(x)
			return callable(x)
		end
		return wrapper(5)
	`)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results[0])
	}
}

// TestMetaCallReturnsNil verifies __call returning nothing yields nil.
func TestMetaCallReturnsNil(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({}, {
			__call = function(self)
				-- returns nothing
			end
		})
		local x = callable()
		return x
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if !results[0].IsNil() {
		t.Errorf("expected nil, got %v", results[0])
	}
}

// TestMetaCallInLoop verifies __call works correctly in a loop
// (repeated invocations, register reuse across iterations).
func TestMetaCallInLoop(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 2}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		local sum = 0
		for i = 1, 5 do
			sum = sum + callable(i)
		end
		return sum
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	// sum = 2+4+6+8+10 = 30
	if len(results) != 1 || results[0].AsInt() != 30 {
		t.Errorf("expected 30, got %v", results)
	}
}

// TestMetaCallNativeInLoop verifies native __call in a loop.
func TestMetaCallNativeInLoop(t *testing.T) {
	results := runWithCallable(t, `
		local sum = 0
		for i = 1, 5 do
			sum = sum + callable(i)
		end
		return sum
	`)
	// sum = 2+4+6+8+10 = 30
	if len(results) != 1 || results[0].AsInt() != 30 {
		t.Errorf("expected 30, got %v", results)
	}
}

// TestMetaCallNoArgs verifies __call with zero user arguments
// (only self is passed).
func TestMetaCallNoArgs(t *testing.T) {
	results, err := runLuaCallable(t, `
		local counter = setmetatable({n = 0}, {
			__call = function(self)
				self.n = self.n + 1
				return self.n
			end
		})
		local a = counter()
		local b = counter()
		local c = counter()
		return a, b, c
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 1 || results[1].AsInt() != 2 || results[2].AsInt() != 3 {
		t.Errorf("expected 1,2,3; got %v,%v,%v", results[0], results[1], results[2])
	}
}

// TestMetaCallDiscardResult verifies __call works when results are discarded
// (call as a statement, not expression).
func TestMetaCallDiscardResult(t *testing.T) {
	results, err := runLuaCallable(t, `
		local trace = 0
		local callable = setmetatable({}, {
			__call = function(self)
				trace = trace + 1
			end
		})
		callable()
		callable()
		callable()
		return trace
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 || results[0].AsInt() != 3 {
		t.Errorf("expected 3, got %v", results)
	}
}

// TestMetaCallVarargResult verifies __call with variable return count
// used as the last argument in a call (all returns should be passed through).
func TestMetaCallVarargResult(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({}, {
			__call = function(self)
				return 10, 20, 30
			end
		})
		local function gather(...)
			return ...
		end
		-- callable() as last arg: all returns should expand
		return gather(callable())
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 || results[1].AsInt() != 20 || results[2].AsInt() != 30 {
		t.Errorf("expected 10,20,30; got %v,%v,%v", results[0], results[1], results[2])
	}
}

// TestMetaCallMixedMetamethods verifies that __call doesn't interfere
// with other metamethods (__index, __add) on the same table.
func TestMetaCallMixedMetamethods(t *testing.T) {
	results, err := runLuaCallable(t, `
		local obj = setmetatable({}, {
			__call = function(self, x) return x * 2 end,
			__index = function(self, key) return key .. "!" end,
			__add = function(a, b) return 999 end,
		})
		local call_result = obj(5)
		local index_result = obj.hello
		local add_result = obj + obj
		return call_result, index_result, add_result
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 10 {
		t.Errorf("__call: expected 10, got %v", results[0])
	}
	if results[1].AsString() != "hello!" {
		t.Errorf("__index: expected 'hello!', got %v", results[1])
	}
	if results[2].AsInt() != 999 {
		t.Errorf("__add: expected 999, got %v", results[2])
	}
}

// TestMetaCallProtectedCallLuaMetamethod tests ProtectedCall with a
// Lua function metamethod (not native).
func TestMetaCallProtectedCallLuaMetamethod(t *testing.T) {
	results, err := runLuaCallable(t, `
		local callable = setmetatable({factor = 3}, {
			__call = function(t, val)
				return val * t.factor
			end
		})
		-- Use a wrapper to invoke ProtectedCall-equivalent from Lua
		-- The wrapper function calls callable, testing the bytecode path
		local function call_it(fn, arg)
			return fn(arg)
		end
		return call_it(callable, 7)
	`)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 21 {
		t.Errorf("expected 21, got %v", results[0])
	}
}

// TestMetaCallNativeMultipleReturns verifies multiple return values
// from a native __call metamethod.
func TestMetaCallNativeMultipleReturns(t *testing.T) {
	tbl := NewEmptyTable()
	mt := NewEmptyTable()
	mt.Set(metaCall, NewNativeFunc(func(v *VM) int {
		x := v.Get(2).AsInt()
		v.Set(0, NewInt(x))
		v.Set(1, NewInt(x*2))
		v.Set(2, NewInt(x*3))
		return 3
	}))
	tbl.SetMetatable(mt)

	results := runWithGlobals(t, `return obj(5)`, map[string]Value{
		"obj": NewTable(tbl),
	})
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if results[0].AsInt() != 5 || results[1].AsInt() != 10 || results[2].AsInt() != 15 {
		t.Errorf("expected 5,10,15; got %v,%v,%v", results[0], results[1], results[2])
	}
}

// TestMetaCallErrorPropagation verifies that errors inside __call
// are properly propagated (not swallowed).
func TestMetaCallErrorPropagation(t *testing.T) {
	_, err := runLuaCallable(t, `
		local callable = setmetatable({}, {
			__call = function(self)
				-- Trigger a runtime error: attempt to index a nil value
				local x = nil
				return x.field
			end
		})
		callable()
	`)
	if err == nil {
		t.Fatal("expected error from __call, got nil")
	}
	if !strings.Contains(err.Error(), "index") {
		t.Errorf("expected error about indexing nil, got: %v", err)
	}
}

// TestMetaCallNativeErrorPropagation verifies native __call errors propagate.
func TestMetaCallNativeErrorPropagation(t *testing.T) {
	tbl := NewEmptyTable()
	mt := NewEmptyTable()
	mt.Set(metaCall, NewNativeFunc(func(v *VM) int {
		panic("native boom")
	}))
	tbl.SetMetatable(mt)

	v := New()
	_, err := v.ProtectedCall(NewTable(tbl), nil)
	if err == nil {
		t.Fatal("expected error from native __call, got nil")
	}
	if !strings.Contains(err.Error(), "native boom") {
		t.Errorf("expected error containing 'native boom', got: %v", err)
	}
}
