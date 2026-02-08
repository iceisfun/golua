package main

import (
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// runLuaReturning compiles and executes Lua source, returning the results.
func runLuaReturning(t *testing.T, source string) []vm.Value {
	t.Helper()
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	stdlib.Open(v)
	results, err := v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return results
}

// === Primary Regression Tests (must fail before fix) ===

// TestMultiReturnComparisonCorruption tests the exact bug pattern:
// multi-return into locals inside a while loop body, followed by an
// equality comparison on the first local. The comparison's boolean
// result (LOADTRUE/LOADFALSE) is written into the register of the
// third local due to freeReg being incorrectly reset to nActVar.
//
// Key: idx==4 is TRUE but ok is the INTEGER 99. If ok gets overwritten
// with the comparison result (true), its type changes from integer to
// boolean — a detectable corruption.
func TestMultiReturnComparisonCorruption(t *testing.T) {
	results := runLuaReturning(t, `
function f()
    return 4, "hello", 99
end

local total = 0
local done = false

while not done do
    local idx, val, ok = f()
    if idx == 4 then
        done = true
    end
    -- ok should be 99 (integer), not true (boolean from idx==4)
    total = ok
end
return total
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 99 {
		t.Errorf("expected ok == 99, got %v — register was corrupted by comparison result",
			results[0])
	}
}

// TestMultiReturnComparisonFalse tests the case where the comparison
// is FALSE. If ok (99) is overwritten with false, the if-ok branch
// won't execute.
func TestMultiReturnComparisonFalse(t *testing.T) {
	results := runLuaReturning(t, `
function f()
    return 5, "hello", 99
end

local result = -1
while true do
    local idx, val, ok = f()
    if idx == 4 then
        -- This won't execute (idx is 5, not 4)
    end
    -- ok should still be 99, not false from the failed comparison
    result = ok
    break
end
return result
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 99 {
		t.Errorf("expected ok == 99, got %v — register was corrupted by comparison result",
			results[0])
	}
}

// TestMultiReturnBooleanExpression ensures comparison temporaries do
// not overwrite locals when the condition uses compound boolean logic.
// Uses distinct integer values to detect corruption.
func TestMultiReturnBooleanExpression(t *testing.T) {
	results := runLuaReturning(t, `
function f()
    return 1, 2, 42
end

local function test()
    local x, y, z = f()
    if x ~= nil and x > 0 then
        -- z should be 42, not a boolean from the comparison
        return z
    end
    return -1
end
return test()
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 42 {
		t.Errorf("expected z == 42, got %v", results[0])
	}
}

// TestNestedExpressionPressure stresses temporary register allocation
// with nested boolean expressions on multi-return locals.
func TestNestedExpressionPressure(t *testing.T) {
	results := runLuaReturning(t, `
function f()
    return 1, 2, 3
end

local function test()
    local a, b, c = f()
    if (a and (b or c)) then
        return b
    end
    return -1
end
return test()
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 2 {
		t.Errorf("expected b == 2, got %v", results[0])
	}
}

// TestControlNoMultiReturn is a control test ensuring that equivalent
// logic without multi-return works correctly. This should always pass
// regardless of the bug, since locals are contiguous from R(0).
func TestControlNoMultiReturn(t *testing.T) {
	results := runLuaReturning(t, `
local function test()
    local a = 1
    local b = 2
    local c = 3
    if a == 1 then
        return c
    end
    return -1
end
return test()
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 3 {
		t.Errorf("expected c == 3, got %v", results[0])
	}
}

// TestForLoopComparisonCorruption tests the same pattern inside a for loop.
func TestForLoopComparisonCorruption(t *testing.T) {
	results := runLuaReturning(t, `
function f()
    return 1, "data", 77
end

local sum = 0
for i = 1, 3 do
    local idx, val, ok = f()
    if idx == 1 then
        sum = sum + ok
    end
end
return sum
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 231 {
		t.Errorf("expected sum == 231 (77*3), got %v", results[0])
	}
}

// TestFourReturnValues tests that the fix works with 4+ return values.
func TestFourReturnValues(t *testing.T) {
	results := runLuaReturning(t, `
function f4()
    return 1, 2, 3, 100
end

local result = -1
while true do
    local a, b, c, d = f4()
    if a == 1 then
        result = d
    end
    break
end
return result
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 100 {
		t.Errorf("expected d == 100, got %v", results[0])
	}
}

// TestRegTopAfterLeaveScope verifies that leaveScope correctly preserves
// local registers when there are gaps from condition temporaries.
func TestRegTopAfterLeaveScope(t *testing.T) {
	results := runLuaReturning(t, `
local result = -1
while true do
    local a = 42
    do
        local b = 99
    end
    -- After leaving the do-block scope, 'a' must not be corrupted.
    -- If freeReg is reset too low, temporaries for the comparison
    -- below could overwrite a's register.
    if a == 42 then
        result = a
    end
    break
end
return result
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	if results[0].AsInt() != 42 {
		t.Errorf("expected result == 42, got %v", results[0])
	}
}
