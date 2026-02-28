package vm

import (
	"math"
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

// Helper to compile and run Lua code
func run(t *testing.T, source string) []Value {
	t.Helper()
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	vm := New()
	results, err := vm.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return results
}

// Helper to run and expect a runtime error
func runError(t *testing.T, source string) error {
	t.Helper()
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	vm := New()
	_, err = vm.Run(proto)
	return err
}

// Helper to run with globals
func runWithGlobals(t *testing.T, source string, globals map[string]Value) []Value {
	t.Helper()
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	vm := New()
	for name, value := range globals {
		vm.SetGlobal(name, value)
	}
	results, err := vm.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return results
}

// Test basic return values
func TestReturnNil(t *testing.T) {
	results := run(t, "return nil")
	if len(results) != 1 || !results[0].IsNil() {
		t.Errorf("expected nil, got %v", results)
	}
}

func TestReturnTrue(t *testing.T) {
	results := run(t, "return true")
	if len(results) != 1 || !results[0].IsBool() || !results[0].AsBool() {
		t.Errorf("expected true, got %v", results)
	}
}

func TestReturnFalse(t *testing.T) {
	results := run(t, "return false")
	if len(results) != 1 || !results[0].IsBool() || results[0].AsBool() {
		t.Errorf("expected false, got %v", results)
	}
}

func TestReturnInteger(t *testing.T) {
	results := run(t, "return 42")
	if len(results) != 1 || results[0].AsInt() != 42 {
		t.Errorf("expected 42, got %v", results)
	}
}

func TestReturnFloat(t *testing.T) {
	results := run(t, "return 3.14")
	if len(results) != 1 || math.Abs(results[0].AsFloat()-3.14) > 0.001 {
		t.Errorf("expected 3.14, got %v", results)
	}
}

func TestReturnString(t *testing.T) {
	results := run(t, `return "hello"`)
	if len(results) != 1 || results[0].AsString() != "hello" {
		t.Errorf("expected 'hello', got %v", results)
	}
}

func TestReturnMultiple(t *testing.T) {
	results := run(t, "return 1, 2, 3")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].AsInt() != 1 || results[1].AsInt() != 2 || results[2].AsInt() != 3 {
		t.Errorf("expected 1, 2, 3, got %v", results)
	}
}

func TestReturnEmpty(t *testing.T) {
	results := run(t, "return")
	if len(results) != 0 {
		t.Errorf("expected no results, got %v", results)
	}
}

// Test local variables
func TestLocalVariable(t *testing.T) {
	results := run(t, "local x = 10; return x")
	if len(results) != 1 || results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results)
	}
}

func TestLocalMultiple(t *testing.T) {
	results := run(t, "local a, b = 1, 2; return a + b")
	if len(results) != 1 || results[0].AsInt() != 3 {
		t.Errorf("expected 3, got %v", results)
	}
}

func TestLocalNilFill(t *testing.T) {
	results := run(t, "local a, b, c = 1, 2; return c")
	if len(results) != 1 || !results[0].IsNil() {
		t.Errorf("expected nil, got %v", results)
	}
}

// Test arithmetic operations
func TestArithmeticAdd(t *testing.T) {
	results := run(t, "return 10 + 20")
	if results[0].AsInt() != 30 {
		t.Errorf("expected 30, got %v", results[0])
	}
}

func TestArithmeticSub(t *testing.T) {
	results := run(t, "return 50 - 30")
	if results[0].AsInt() != 20 {
		t.Errorf("expected 20, got %v", results[0])
	}
}

func TestArithmeticMul(t *testing.T) {
	results := run(t, "return 6 * 7")
	if results[0].AsInt() != 42 {
		t.Errorf("expected 42, got %v", results[0])
	}
}

func TestArithmeticDiv(t *testing.T) {
	results := run(t, "return 10 / 4")
	if results[0].AsFloat() != 2.5 {
		t.Errorf("expected 2.5, got %v", results[0])
	}
}

func TestArithmeticIdiv(t *testing.T) {
	results := run(t, "return 10 // 3")
	if results[0].AsInt() != 3 {
		t.Errorf("expected 3, got %v", results[0])
	}
}

func TestArithmeticMod(t *testing.T) {
	results := run(t, "return 17 % 5")
	if results[0].AsInt() != 2 {
		t.Errorf("expected 2, got %v", results[0])
	}
}

func TestArithmeticPow(t *testing.T) {
	results := run(t, "return 2 ^ 10")
	if results[0].AsFloat() != 1024 {
		t.Errorf("expected 1024, got %v", results[0])
	}
}

func TestArithmeticNegate(t *testing.T) {
	results := run(t, "return -42")
	if results[0].AsInt() != -42 {
		t.Errorf("expected -42, got %v", results[0])
	}
}

// Test bitwise operations
func TestBitwiseAnd(t *testing.T) {
	results := run(t, "return 0xF0 & 0x0F")
	if results[0].AsInt() != 0 {
		t.Errorf("expected 0, got %v", results[0])
	}
}

func TestBitwiseOr(t *testing.T) {
	results := run(t, "return 0xF0 | 0x0F")
	if results[0].AsInt() != 0xFF {
		t.Errorf("expected 255, got %v", results[0])
	}
}

func TestBitwiseXor(t *testing.T) {
	results := run(t, "return 0xFF ~ 0x0F")
	if results[0].AsInt() != 0xF0 {
		t.Errorf("expected 240, got %v", results[0])
	}
}

func TestBitwiseNot(t *testing.T) {
	results := run(t, "return ~0")
	if results[0].AsInt() != -1 {
		t.Errorf("expected -1, got %v", results[0])
	}
}

func TestBitwiseShl(t *testing.T) {
	results := run(t, "return 1 << 4")
	if results[0].AsInt() != 16 {
		t.Errorf("expected 16, got %v", results[0])
	}
}

func TestBitwiseShr(t *testing.T) {
	results := run(t, "return 16 >> 2")
	if results[0].AsInt() != 4 {
		t.Errorf("expected 4, got %v", results[0])
	}
}

// Test logical operations
func TestLogicalNot(t *testing.T) {
	tests := []struct {
		src    string
		expect bool
	}{
		{"return not nil", true},
		{"return not false", true},
		{"return not true", false},
		{"return not 0", false},
		{"return not ''", false},
	}
	for _, tc := range tests {
		results := run(t, tc.src)
		if results[0].AsBool() != tc.expect {
			t.Errorf("%s: expected %v, got %v", tc.src, tc.expect, results[0])
		}
	}
}

func TestLogicalAnd(t *testing.T) {
	tests := []struct {
		src    string
		expect int64
	}{
		{"return 1 and 2", 2},
		{"return false and 2", 0}, // returns false, which is 0
		{"return nil and 2", 0},   // returns nil
	}
	for _, tc := range tests {
		results := run(t, tc.src)
		if tc.expect == 0 {
			if results[0].ToBool() {
				t.Errorf("%s: expected falsy, got %v", tc.src, results[0])
			}
		} else if results[0].AsInt() != tc.expect {
			t.Errorf("%s: expected %d, got %v", tc.src, tc.expect, results[0])
		}
	}
}

func TestLogicalOr(t *testing.T) {
	tests := []struct {
		src      string
		expectI  int64
		expectFalsy bool
	}{
		{"return 1 or 2", 1, false},
		{"return false or 2", 2, false},
		{"return nil or false", 0, true},
	}
	for _, tc := range tests {
		results := run(t, tc.src)
		if tc.expectFalsy {
			if results[0].ToBool() {
				t.Errorf("%s: expected falsy, got %v", tc.src, results[0])
			}
		} else if results[0].AsInt() != tc.expectI {
			t.Errorf("%s: expected %d, got %v", tc.src, tc.expectI, results[0])
		}
	}
}

// Test comparison operations
func TestComparisonEq(t *testing.T) {
	tests := []struct {
		src    string
		expect bool
	}{
		{"return 1 == 1", true},
		{"return 1 == 2", false},
		{"return 'a' == 'a'", true},
		{"return 'a' == 'b'", false},
		{"return nil == nil", true},
		{"return 1 == 1.0", true}, // int-float comparison
	}
	for _, tc := range tests {
		results := run(t, tc.src)
		if results[0].AsBool() != tc.expect {
			t.Errorf("%s: expected %v, got %v", tc.src, tc.expect, results[0])
		}
	}
}

func TestComparisonNe(t *testing.T) {
	results := run(t, "return 1 ~= 2")
	if !results[0].AsBool() {
		t.Errorf("expected true, got %v", results[0])
	}
}

func TestComparisonLt(t *testing.T) {
	tests := []struct {
		src    string
		expect bool
	}{
		{"return 1 < 2", true},
		{"return 2 < 1", false},
		{"return 1 < 1", false},
		{"return 'a' < 'b'", true},
	}
	for _, tc := range tests {
		results := run(t, tc.src)
		if results[0].AsBool() != tc.expect {
			t.Errorf("%s: expected %v, got %v", tc.src, tc.expect, results[0])
		}
	}
}

func TestComparisonLe(t *testing.T) {
	tests := []struct {
		src    string
		expect bool
	}{
		{"return 1 <= 2", true},
		{"return 1 <= 1", true},
		{"return 2 <= 1", false},
	}
	for _, tc := range tests {
		results := run(t, tc.src)
		if results[0].AsBool() != tc.expect {
			t.Errorf("%s: expected %v, got %v", tc.src, tc.expect, results[0])
		}
	}
}

func TestComparisonGt(t *testing.T) {
	results := run(t, "return 2 > 1")
	if !results[0].AsBool() {
		t.Errorf("expected true, got %v", results[0])
	}
}

func TestComparisonGe(t *testing.T) {
	results := run(t, "return 2 >= 2")
	if !results[0].AsBool() {
		t.Errorf("expected true, got %v", results[0])
	}
}

// Test string operations
func TestStringConcat(t *testing.T) {
	results := run(t, `return "hello" .. " " .. "world"`)
	if results[0].AsString() != "hello world" {
		t.Errorf("expected 'hello world', got %v", results[0])
	}
}

func TestStringLength(t *testing.T) {
	results := run(t, `return #"hello"`)
	if results[0].AsInt() != 5 {
		t.Errorf("expected 5, got %v", results[0])
	}
}

func TestConcatNumbers(t *testing.T) {
	results := run(t, `return 1 .. 2 .. 3`)
	if results[0].AsString() != "123" {
		t.Errorf("expected '123', got %v", results[0])
	}
}

// Test control flow
func TestIfTrue(t *testing.T) {
	results := run(t, `
		local x = 0
		if true then x = 1 end
		return x
	`)
	if results[0].AsInt() != 1 {
		t.Errorf("expected 1, got %v", results[0])
	}
}

func TestIfFalse(t *testing.T) {
	results := run(t, `
		local x = 0
		if false then x = 1 end
		return x
	`)
	if results[0].AsInt() != 0 {
		t.Errorf("expected 0, got %v", results[0])
	}
}

func TestIfElse(t *testing.T) {
	results := run(t, `
		local x
		if false then x = 1 else x = 2 end
		return x
	`)
	if results[0].AsInt() != 2 {
		t.Errorf("expected 2, got %v", results[0])
	}
}

func TestIfElseif(t *testing.T) {
	results := run(t, `
		local x
		if false then x = 1
		elseif true then x = 2
		else x = 3
		end
		return x
	`)
	if results[0].AsInt() != 2 {
		t.Errorf("expected 2, got %v", results[0])
	}
}

func TestWhileLoop(t *testing.T) {
	results := run(t, `
		local i, sum = 1, 0
		while i <= 5 do
			sum = sum + i
			i = i + 1
		end
		return sum
	`)
	if results[0].AsInt() != 15 {
		t.Errorf("expected 15, got %v", results[0])
	}
}

func TestRepeatUntil(t *testing.T) {
	results := run(t, `
		local i, sum = 1, 0
		repeat
			sum = sum + i
			i = i + 1
		until i > 5
		return sum
	`)
	if results[0].AsInt() != 15 {
		t.Errorf("expected 15, got %v", results[0])
	}
}

func TestForNumeric(t *testing.T) {
	results := run(t, `
		local sum = 0
		for i = 1, 5 do
			sum = sum + i
		end
		return sum
	`)
	if results[0].AsInt() != 15 {
		t.Errorf("expected 15, got %v", results[0])
	}
}

func TestForNumericWithStep(t *testing.T) {
	results := run(t, `
		local sum = 0
		for i = 10, 2, -2 do
			sum = sum + i
		end
		return sum
	`)
	// 10 + 8 + 6 + 4 + 2 = 30
	if results[0].AsInt() != 30 {
		t.Errorf("expected 30, got %v", results[0])
	}
}

func TestBreak(t *testing.T) {
	results := run(t, `
		local sum = 0
		for i = 1, 10 do
			if i > 5 then break end
			sum = sum + i
		end
		return sum
	`)
	if results[0].AsInt() != 15 {
		t.Errorf("expected 15, got %v", results[0])
	}
}

// Test tables
func TestTableEmpty(t *testing.T) {
	results := run(t, "local t = {}; return t")
	if !results[0].IsTable() {
		t.Errorf("expected table, got %v", results[0])
	}
}

func TestTableArray(t *testing.T) {
	results := run(t, "local t = {1, 2, 3}; return t[1], t[2], t[3]")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].AsInt() != 1 || results[1].AsInt() != 2 || results[2].AsInt() != 3 {
		t.Errorf("expected 1, 2, 3, got %v", results)
	}
}

func TestTableRecord(t *testing.T) {
	results := run(t, `local t = {x = 10, y = 20}; return t.x, t.y`)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].AsInt() != 10 || results[1].AsInt() != 20 {
		t.Errorf("expected 10, 20, got %v", results)
	}
}

func TestTableBracketKey(t *testing.T) {
	results := run(t, `local t = {["hello"] = "world"}; return t["hello"]`)
	if results[0].AsString() != "world" {
		t.Errorf("expected 'world', got %v", results[0])
	}
}

func TestTableLength(t *testing.T) {
	results := run(t, "local t = {1, 2, 3, 4, 5}; return #t")
	if results[0].AsInt() != 5 {
		t.Errorf("expected 5, got %v", results[0])
	}
}

func TestTableAssignment(t *testing.T) {
	results := run(t, `
		local t = {}
		t.x = 10
		t["y"] = 20
		t[1] = 30
		return t.x, t.y, t[1]
	`)
	if results[0].AsInt() != 10 || results[1].AsInt() != 20 || results[2].AsInt() != 30 {
		t.Errorf("expected 10, 20, 30, got %v", results)
	}
}

// Test functions
func TestFunctionDefinition(t *testing.T) {
	results := run(t, `
		local function add(a, b)
			return a + b
		end
		return add(3, 4)
	`)
	if results[0].AsInt() != 7 {
		t.Errorf("expected 7, got %v", results[0])
	}
}

func TestFunctionExpression(t *testing.T) {
	results := run(t, `
		local add = function(a, b) return a + b end
		return add(3, 4)
	`)
	if results[0].AsInt() != 7 {
		t.Errorf("expected 7, got %v", results[0])
	}
}

func TestFunctionMultipleReturns(t *testing.T) {
	results := run(t, `
		local function swap(a, b)
			return b, a
		end
		local x, y = swap(1, 2)
		return x, y
	`)
	if results[0].AsInt() != 2 || results[1].AsInt() != 1 {
		t.Errorf("expected 2, 1, got %v", results)
	}
}

func TestFunctionRecursion(t *testing.T) {
	results := run(t, `
		local function fib(n)
			if n <= 1 then return n end
			return fib(n - 1) + fib(n - 2)
		end
		return fib(10)
	`)
	if results[0].AsInt() != 55 {
		t.Errorf("expected 55 (fib(10)), got %v", results[0])
	}
}

// Simpler recursion tests to isolate issues
func TestSimpleRecursion(t *testing.T) {
	// Base case only
	results := run(t, `
		local function f(n)
			if n <= 0 then return 42 end
			return f(n - 1)
		end
		return f(0)
	`)
	if results[0].AsInt() != 42 {
		t.Errorf("f(0): expected 42, got %v", results[0])
	}
}

func TestRecursionOneLevel(t *testing.T) {
	results := run(t, `
		local function f(n)
			if n <= 0 then return 42 end
			return f(n - 1)
		end
		return f(1)
	`)
	if results[0].AsInt() != 42 {
		t.Errorf("f(1): expected 42, got %v", results[0])
	}
}

func TestRecursionTwoLevels(t *testing.T) {
	results := run(t, `
		local function f(n)
			if n <= 0 then return 42 end
			return f(n - 1)
		end
		return f(2)
	`)
	if results[0].AsInt() != 42 {
		t.Errorf("f(2): expected 42, got %v", results[0])
	}
}

func TestFibBaseCase(t *testing.T) {
	results := run(t, `
		local function fib(n)
			if n <= 1 then return n end
			return fib(n - 1) + fib(n - 2)
		end
		return fib(1)
	`)
	if results[0].AsInt() != 1 {
		t.Errorf("fib(1): expected 1, got %v", results[0])
	}
}

func TestFibTwoWithLocals(t *testing.T) {
	results := run(t, `
		local function fib(n)
			if n <= 1 then return n end
			local a = fib(n - 1)
			local b = fib(n - 2)
			return a + b
		end
		return fib(2)
	`)
	if results[0].AsInt() != 1 {
		t.Errorf("fib(2): expected 1, got %v", results[0])
	}
}

func TestTwoCallsToSameFunction(t *testing.T) {
	results := run(t, `
		local function f() return 10 end
		local a = f()
		local b = f()
		return a + b
	`)
	if results[0].AsInt() != 20 {
		t.Errorf("expected 20, got %v", results[0])
	}
}

func TestTwoRecursiveCalls(t *testing.T) {
	results := run(t, `
		local function f(n)
			if n <= 0 then return 1 end
			return f(n - 1)
		end
		local a = f(1)
		local b = f(1)
		return a + b
	`)
	if results[0].AsInt() != 2 {
		t.Errorf("expected 2, got %v", results[0])
	}
}

func TestTwoRecursiveCallsInsideFunction(t *testing.T) {
	results := run(t, `
		local function f(n)
			if n <= 0 then return 1 end
			local a = f(n - 1)
			local b = f(n - 1)
			return a + b
		end
		return f(1)
	`)
	if results[0].AsInt() != 2 {
		t.Errorf("expected 2, got %v", results[0])
	}
}

func TestFunctionVararg(t *testing.T) {
	results := run(t, `
		local function sum(...)
			local s = 0
			local args = {...}
			for i = 1, #args do
				s = s + args[i]
			end
			return s
		end
		return sum(1, 2, 3, 4, 5)
	`)
	if results[0].AsInt() != 15 {
		t.Errorf("expected 15, got %v", results[0])
	}
}

// Test closures
func TestClosureCapture(t *testing.T) {
	results := run(t, `
		local function counter()
			local count = 0
			return function()
				count = count + 1
				return count
			end
		end
		local c = counter()
		return c(), c(), c()
	`)
	if results[0].AsInt() != 1 || results[1].AsInt() != 2 || results[2].AsInt() != 3 {
		t.Errorf("expected 1, 2, 3, got %v", results)
	}
}

func TestClosureNested(t *testing.T) {
	results := run(t, `
		local function make_adder(x)
			return function(y)
				return x + y
			end
		end
		local add5 = make_adder(5)
		return add5(10)
	`)
	if results[0].AsInt() != 15 {
		t.Errorf("expected 15, got %v", results[0])
	}
}

// Test globals
func TestGlobalAccess(t *testing.T) {
	results := runWithGlobals(t, "return x", map[string]Value{
		"x": NewInt(42),
	})
	if results[0].AsInt() != 42 {
		t.Errorf("expected 42, got %v", results[0])
	}
}

func TestGlobalAssignment(t *testing.T) {
	block, _ := parser.Parse("<test>", "x = 100; return x")
	proto, _ := compiler.Compile("<test>", block)
	vm := New()
	results, err := vm.Run(proto)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].AsInt() != 100 {
		t.Errorf("expected 100, got %v", results[0])
	}
	// Verify it's in globals
	if vm.GetGlobal("x").AsInt() != 100 {
		t.Error("global not set")
	}
}

// Test method calls
func TestMethodCall(t *testing.T) {
	results := run(t, `
		local obj = {
			value = 10,
			getValue = function(self) return self.value end
		}
		return obj:getValue()
	`)
	if results[0].AsInt() != 10 {
		t.Errorf("expected 10, got %v", results[0])
	}
}

// Test native functions
func TestNativeFunction(t *testing.T) {
	block, _ := parser.Parse("<test>", "return add(3, 4)")
	proto, _ := compiler.Compile("<test>", block)
	vm := New()

	// Register a native function
	vm.SetGlobal("add", NewNativeFunc(func(vm *VM) int {
		a := vm.Get(1)
		b := vm.Get(2)
		vm.stack[vm.Base()] = NewInt(a.AsInt() + b.AsInt())
		return 1
	}))

	results, err := vm.Run(proto)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].AsInt() != 7 {
		t.Errorf("expected 7, got %v", results[0])
	}
}

// Test for-in loop with native iterator
func TestForInWithPairs(t *testing.T) {
	block, _ := parser.Parse("<test>", `
		local sum = 0
		for k, v in pairs(t) do
			sum = sum + v
		end
		return sum
	`)
	proto, _ := compiler.Compile("<test>", block)
	vm := New()

	// Create a table
	tbl := NewEmptyTable()
	tbl.SetInt(1, NewInt(10))
	tbl.SetInt(2, NewInt(20))
	tbl.SetInt(3, NewInt(30))
	vm.SetGlobal("t", NewTable(tbl))

	// Simple pairs implementation
	vm.SetGlobal("pairs", NewNativeFunc(func(vm *VM) int {
		tbl := vm.Get(1).AsTable()
		// Return iterator function, table, nil
		iterFunc := NewNativeFunc(func(vm *VM) int {
			t := vm.Get(1).AsTable()
			k := vm.Get(2)
			nextK, nextV, _ := t.Next(k)
			vm.stack[vm.Base()] = nextK
			vm.stack[vm.Base()+1] = nextV
			return 2
		})
		vm.stack[vm.Base()] = NewNativeFunc(iterFunc.AsNativeFunc())
		vm.stack[vm.Base()+1] = NewTable(tbl)
		vm.stack[vm.Base()+2] = Nil
		return 3
	}))

	results, err := vm.Run(proto)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].AsInt() != 60 {
		t.Errorf("expected 60, got %v", results[0])
	}
}

// Test error cases
func TestArithmeticOnNil(t *testing.T) {
	err := runError(t, "return nil + 1")
	if err == nil || !strings.Contains(err.Error(), "arithmetic") {
		t.Errorf("expected arithmetic error, got %v", err)
	}
}

func TestCompareNilWithNumber(t *testing.T) {
	err := runError(t, "return nil < 1")
	if err == nil || !strings.Contains(err.Error(), "compare") {
		t.Errorf("expected comparison error, got %v", err)
	}
}

func TestCallNonFunction(t *testing.T) {
	err := runError(t, "local x = 1; return x()")
	if err == nil || !strings.Contains(err.Error(), "call") {
		t.Errorf("expected call error, got %v", err)
	}
}

func TestIndexNonTable(t *testing.T) {
	err := runError(t, "local x = 1; return x.foo")
	if err == nil || !strings.Contains(err.Error(), "index") {
		t.Errorf("expected index error, got %v", err)
	}
}

// Test complex expressions
func TestComplexExpression(t *testing.T) {
	results := run(t, `
		local a, b, c = 1, 2, 3
		return (a + b) * c - a / b
	`)
	// (1 + 2) * 3 - 1 / 2 = 9 - 0.5 = 8.5
	if math.Abs(results[0].AsFloat()-8.5) > 0.001 {
		t.Errorf("expected 8.5, got %v", results[0])
	}
}

// Test do-end blocks and scoping
func TestDoBlock(t *testing.T) {
	results := run(t, `
		local x = 1
		do
			local x = 2
		end
		return x
	`)
	if results[0].AsInt() != 1 {
		t.Errorf("expected 1, got %v", results[0])
	}
}

// Test assignment targets
func TestMultipleAssignment(t *testing.T) {
	results := run(t, `
		local a, b, c
		a, b, c = 1, 2, 3
		return a, b, c
	`)
	if results[0].AsInt() != 1 || results[1].AsInt() != 2 || results[2].AsInt() != 3 {
		t.Errorf("expected 1, 2, 3, got %v", results)
	}
}

func TestSwapAssignment(t *testing.T) {
	results := run(t, `
		local a, b = 1, 2
		a, b = b, a
		return a, b
	`)
	if results[0].AsInt() != 2 || results[1].AsInt() != 1 {
		t.Errorf("expected 2, 1, got %v", results)
	}
}

// Test edge cases
func TestEmptyProgram(t *testing.T) {
	results := run(t, "")
	if len(results) != 0 {
		t.Errorf("expected no results, got %v", results)
	}
}

func TestOnlyComments(t *testing.T) {
	results := run(t, "-- just a comment")
	if len(results) != 0 {
		t.Errorf("expected no results, got %v", results)
	}
}

// Tail call optimization test
func TestTailCallOptimization(t *testing.T) {
	// This would stack overflow without tail call optimization
	results := run(t, `
		local function count(n, acc)
			if n <= 0 then return acc end
			return count(n - 1, acc + 1)
		end
		return count(1000, 0)
	`)
	if results[0].AsInt() != 1000 {
		t.Errorf("expected 1000, got %v", results[0])
	}
}

// ──────────────────────────────────────────────────────────────
// Stack management: vm.top restoration and native call stale data
// ──────────────────────────────────────────────────────────────

// runWithStdlib creates a VM with essential stdlib functions for stack tests.
func runWithStdlib(t *testing.T, source string) []Value {
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
	// table.concat
	tblLib := NewEmptyTable()
	tblLib.SetString("concat", NewNativeFunc(func(v *VM) int {
		tbl := v.Get(1).AsTable()
		sep := ""
		if !v.Get(2).IsNil() {
			sep = v.Get(2).AsString()
		}
		length := tbl.Len()
		i := int64(1)
		if !v.Get(3).IsNil() {
			i = v.Get(3).AsInt()
		}
		j := int64(length)
		if !v.Get(4).IsNil() {
			j = v.Get(4).AsInt()
		}
		var parts []string
		for idx := i; idx <= j; idx++ {
			val := tbl.Get(NewInt(idx))
			parts = append(parts, val.AsString())
		}
		v.Set(0, NewString(strings.Join(parts, sep)))
		return 1
	}))
	v.SetGlobal("table", NewTable(tblLib))

	// pcall
	v.SetGlobal("pcall", NewNativeFunc(func(v *VM) int {
		fn := v.Get(1)
		argc := v.ArgCount()
		args := make([]Value, argc-1)
		for i := 2; i <= argc; i++ {
			args[i-2] = v.Get(i)
		}
		results, err := v.ProtectedCall(fn, args)
		if err != nil {
			v.Set(0, False)
			if le, ok := err.(*LuaError); ok {
				v.Set(1, le.Value)
			} else {
				v.Set(1, NewString(err.Error()))
			}
			return 2
		}
		v.Set(0, True)
		for i, r := range results {
			v.Set(i+1, r)
		}
		return 1 + len(results)
	}))

	// tostring (with __tostring metamethod support)
	v.SetGlobal("tostring", NewNativeFunc(func(v *VM) int {
		val := v.Get(1)
		if val.IsTable() {
			if mt := val.AsTable().Metatable(); mt != nil {
				if ts := mt.Get(NewString("__tostring")); !ts.IsNil() {
					results, err := v.ProtectedCall(ts, []Value{val})
					if err != nil {
						panic(err)
					}
					if len(results) > 0 {
						v.Set(0, results[0])
					} else {
						v.Set(0, NewString("nil"))
					}
					return 1
				}
			}
		}
		v.Set(0, NewString(val.String()))
		return 1
	}))

	// setmetatable
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

	// ipairs
	v.SetGlobal("ipairs", NewNativeFunc(func(v *VM) int {
		tbl := v.Get(1).AsTable()
		idx := int64(0)
		v.Set(0, NewNativeFunc(func(v *VM) int {
			idx++
			val := tbl.Get(NewInt(idx))
			if val.IsNil() {
				return 0
			}
			v.Set(0, NewInt(idx))
			v.Set(1, val)
			return 2
		}))
		v.Set(1, v.Get(1))
		v.Set(2, NewInt(0))
		return 3
	}))

	// type
	v.SetGlobal("type", NewNativeFunc(func(v *VM) int {
		v.Set(0, NewString(v.Get(1).Type()))
		return 1
	}))

	results, runErr := v.Run(proto)
	if runErr != nil {
		t.Fatalf("runtime error: %v", runErr)
	}
	return results
}

// TestStackPcallThenConcat verifies that pcall doesn't leave stale data
// visible to table.concat's optional arguments via tail call.
func TestStackPcallThenConcat(t *testing.T) {
	results := runWithStdlib(t, `
		local function f()
			local ok = pcall(function() return 1 end)
			return table.concat({"a", "b"}, ".")
		end
		return f()
	`)
	if len(results) != 1 || results[0].AsString() != "a.b" {
		t.Fatalf("expected 'a.b', got %v", results)
	}
}

// TestStackRecursiveTostring verifies deeply nested __tostring calls work
// when tostring uses ProtectedCall and table.concat is called via tail call.
func TestStackRecursiveTostring(t *testing.T) {
	results := runWithStdlib(t, `
		local function make_node(name, children)
			return setmetatable({name = name, children = children or {}}, {
				__tostring = function(self)
					local parts = {self.name}
					for _, child in ipairs(self.children) do
						parts[#parts + 1] = tostring(child)
					end
					return table.concat(parts, ".")
				end
			})
		end
		local tree = make_node("a", {
			make_node("b", {make_node("d"), make_node("e")}),
			make_node("c")
		})
		return tostring(tree)
	`)
	if len(results) != 1 || results[0].AsString() != "a.b.d.e.c" {
		t.Fatalf("expected 'a.b.d.e.c', got %v", results)
	}
}

// TestStackInlineNestedCalls verifies that inline nested function calls
// (which use c=0 CALL + SETLIST bytecode patterns) don't corrupt vm.top.
func TestStackInlineNestedCalls(t *testing.T) {
	results := runWithStdlib(t, `
		local function make_node(name, children)
			return setmetatable({name = name, children = children or {}}, {
				__tostring = function(self)
					local parts = {self.name}
					for _, child in ipairs(self.children) do
						parts[#parts + 1] = tostring(child)
					end
					return table.concat(parts, ".")
				end
			})
		end
		-- Inline creation: make_node("a", {make_node("b", {make_node("d")})})
		-- This generates c=0 CALL + SETLIST bytecode that lowers vm.top
		local parent = make_node("a", {make_node("b")})
		local p = tostring(parent)
		local deep = make_node("a2", {make_node("b2", {make_node("d2")})})
		local d = tostring(deep)
		return p, d
	`)
	if len(results) != 2 || results[0].AsString() != "a.b" || results[1].AsString() != "a2.b2.d2" {
		t.Fatalf("expected 'a.b','a2.b2.d2', got %v", results)
	}
}

// TestStackProtectedCallZeroArgs verifies that ProtectedCall with zero args
// correctly reports ArgCount() == 0 to the called native function.
func TestStackProtectedCallZeroArgs(t *testing.T) {
	block, err := parser.Parse("<test>", "return pcall(get_argc)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := New()
	v.SetGlobal("pcall", NewNativeFunc(func(v *VM) int {
		fn := v.Get(1)
		argc := v.ArgCount()
		args := make([]Value, argc-1)
		for i := 2; i <= argc; i++ {
			args[i-2] = v.Get(i)
		}
		results, err := v.ProtectedCall(fn, args)
		if err != nil {
			v.Set(0, False)
			v.Set(1, NewString(err.Error()))
			return 2
		}
		v.Set(0, True)
		for i, r := range results {
			v.Set(i+1, r)
		}
		return 1 + len(results)
	}))
	v.SetGlobal("get_argc", NewNativeFunc(func(v *VM) int {
		v.Set(0, NewInt(int64(v.ArgCount())))
		return 1
	}))
	results, err := v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	// pcall(get_argc) → true, 0 (zero args passed to get_argc)
	if len(results) < 2 || !results[0].AsBool() || results[1].AsInt() != 0 {
		t.Fatalf("expected true,0 got %v", results)
	}
}

// TestStackNestedPcall verifies that nested pcall (ProtectedCall calling
// ProtectedCall) correctly manages stack frames.
func TestStackNestedPcall(t *testing.T) {
	results := runWithStdlib(t, `
		local ok, r = pcall(function()
			local ok2, r2 = pcall(function()
				return table.concat({"x", "y"}, "-")
			end)
			return r2 .. "!" .. table.concat({"a", "b"}, ".")
		end)
		return ok, r
	`)
	if len(results) < 2 || !results[0].AsBool() || results[1].AsString() != "x-y!a.b" {
		t.Fatalf("expected true,'x-y!a.b', got %v", results)
	}
}

// TestUnpackLargeTable verifies that table.unpack can handle tables larger
// than MaxStack (256) without panicking.
func TestUnpackLargeTable(t *testing.T) {
	// Create a table with 500 elements and try to unpack it.
	// This should grow the stack rather than panic with index out of range.
	v := New()

	// Register table.unpack as a native function
	tblLib := NewEmptyTable()
	tblLib.SetString("unpack", NewNativeFunc(func(vm *VM) int {
		tbl := vm.Get(1).AsTable()
		length := tbl.Len()
		i := 1
		if !vm.Get(2).IsNil() {
			i = int(vm.Get(2).AsInt())
		}
		j := length
		if !vm.Get(3).IsNil() {
			j = int(vm.Get(3).AsInt())
		}
		n := j - i + 1
		if n > 0 {
			vm.EnsureStack(vm.Base() + n)
		}
		count := 0
		for idx := i; idx <= j; idx++ {
			vm.Set(count, tbl.Get(NewInt(int64(idx))))
			count++
		}
		return count
	}))
	v.SetGlobal("table", NewTable(tblLib))

	// Build a 500-element table
	big := NewEmptyTable()
	for i := 1; i <= 500; i++ {
		big.SetInt(i, NewInt(int64(i)))
	}
	v.SetGlobal("big", NewTable(big))

	// This should NOT panic
	block, err := parser.Parse("<test>", `
		local vals = {table.unpack(big)}
		local sum = 0
		for i = 1, #vals do
			sum = sum + vals[i]
		end
		return sum
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// Catch panics — the bug is a panic, not an error
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unpack panicked (stack overflow): %v", r)
		}
	}()

	results, err := v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) < 1 {
		t.Fatal("expected a result")
	}
	sum := results[0].AsInt()
	if sum != 125250 {
		t.Fatalf("expected sum 125250, got %d", sum)
	}
}

// TestStringByteLarge verifies that string.byte can return more than MaxStack
// (256) values without panicking due to stack overflow.
func TestStringByteLarge(t *testing.T) {
	v := New()

	// Register string.byte as a native function
	strLib := NewEmptyTable()
	strLib.SetString("byte", NewNativeFunc(func(vm *VM) int {
		s := vm.Get(1).AsString()
		i := int64(1)
		if !vm.Get(2).IsNil() {
			i = vm.Get(2).AsInt()
		}
		j := i
		if !vm.Get(3).IsNil() {
			j = vm.Get(3).AsInt()
		}
		start := int(i)
		end := int(j)
		if start < 1 {
			start = 1
		}
		if end > len(s) {
			end = len(s)
		}
		n := end - start + 1
		if n > 0 {
			vm.EnsureStack(vm.Base() + n)
		}
		count := 0
		for idx := start; idx <= end; idx++ {
			vm.Set(count, NewInt(int64(s[idx-1])))
			count++
		}
		return count
	}))
	v.SetGlobal("string", NewTable(strLib))

	// Build a 500-byte string
	s := strings.Repeat("A", 500)
	v.SetGlobal("s", NewString(s))

	block, err := parser.Parse("<test>", `
		local vals = {string.byte(s, 1, #s)}
		local sum = 0
		for i = 1, #vals do
			sum = sum + vals[i]
		end
		return sum, #vals
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("string.byte panicked (stack overflow): %v", r)
		}
	}()

	results, err := v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if len(results) < 2 {
		t.Fatal("expected 2 results")
	}
	// 'A' = 65, 500 * 65 = 32500
	if results[0].AsInt() != 32500 {
		t.Fatalf("expected sum 32500, got %d", results[0].AsInt())
	}
	if results[1].AsInt() != 500 {
		t.Fatalf("expected 500 values, got %d", results[1].AsInt())
	}
}

// Value and Table unit tests live in value_test.go and table_test.go respectively.
