package main

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// runLuaAndOr compiles and runs Lua source, capturing print output.
func runLuaAndOr(t *testing.T, source string) string {
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
	var buf strings.Builder
	v.SetGlobal("print", vm.NewNativeFunc(func(v *vm.VM) int {
		for i := 1; i <= v.ArgCount(); i++ {
			if i > 1 {
				buf.WriteByte('\t')
			}
			buf.WriteString(v.Get(i).AsString())
		}
		buf.WriteByte('\n')
		return 0
	}))
	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return strings.TrimSpace(buf.String())
}

// TestAndOrTernaryInFormatMultipleCalls reproduces a bug where calling a
// function that uses `v and "Y" or "N"` multiple times as arguments to
// string.format returns the raw boolean value instead of "Y"/"N" for
// the third (or later) call.
//
// The user's original code:
//   local function yn(v) return v and "Y" or "N" end
//   print(string.format("BC:%s BO:%s Shout:%s", yn(bc), yn(bo), yn(sh)))
// Output was: BC:Y BO:Y Shout:true  (expected: BC:Y BO:Y Shout:Y)
func TestAndOrTernaryInFormatMultipleCalls(t *testing.T) {
	out := runLuaAndOr(t, `
local a = true
local b = true
local c = true

local function yn(v) return v and "Y" or "N" end
print(string.format("A:%s B:%s C:%s", yn(a), yn(b), yn(c)))
`)
	if out != "A:Y B:Y C:Y" {
		t.Errorf("expected 'A:Y B:Y C:Y', got %q", out)
	}
}

// TestAndOrTernaryThreeCallsNoFormat — same bug but without string.format,
// just passing three yn() calls to print directly.
func TestAndOrTernaryThreeCallsNoFormat(t *testing.T) {
	out := runLuaAndOr(t, `
local a = true
local b = true
local c = true

local function yn(v) return v and "Y" or "N" end
print(yn(a), yn(b), yn(c))
`)
	if out != "Y\tY\tY" {
		t.Errorf("expected 'Y\\tY\\tY', got %q", out)
	}
}

// TestAndOrTernaryMixedValues tests with a mix of true/false/nil values.
func TestAndOrTernaryMixedValues(t *testing.T) {
	out := runLuaAndOr(t, `
local a = true
local b = false
local c = true

local function yn(v) return v and "Y" or "N" end
print(string.format("A:%s B:%s C:%s", yn(a), yn(b), yn(c)))
`)
	if out != "A:Y B:N C:Y" {
		t.Errorf("expected 'A:Y B:N C:Y', got %q", out)
	}
}

// TestAndOrTernaryFourCalls tests with four calls to increase register pressure.
func TestAndOrTernaryFourCalls(t *testing.T) {
	out := runLuaAndOr(t, `
local a = true
local b = true
local c = true
local d = true

local function yn(v) return v and "Y" or "N" end
print(string.format("A:%s B:%s C:%s D:%s", yn(a), yn(b), yn(c), yn(d)))
`)
	if out != "A:Y B:Y C:Y D:Y" {
		t.Errorf("expected 'A:Y B:Y C:Y D:Y', got %q", out)
	}
}

// TestAndOrTernaryInlineNoFunction tests and/or ternary without wrapping
// in a function — directly as string.format arguments.
func TestAndOrTernaryInlineNoFunction(t *testing.T) {
	out := runLuaAndOr(t, `
local a = true
local b = true
local c = true
print(string.format("A:%s B:%s C:%s",
    a and "Y" or "N",
    b and "Y" or "N",
    c and "Y" or "N"))
`)
	if out != "A:Y B:Y C:Y" {
		t.Errorf("expected 'A:Y B:Y C:Y', got %q", out)
	}
}

// TestAndOrWithMethodCallArgs mimics the original bug more closely:
// locals assigned from function returns, then used in and/or ternary.
func TestAndOrWithMethodCallArgs(t *testing.T) {
	out := runLuaAndOr(t, `
local function has_state(name)
    return true
end

local bc = has_state("Battle Command")
local bo = has_state("Battle Orders")
local sh = has_state("Shout")

local function yn(v) return v and "Y" or "N" end
print(string.format("BC:%s BO:%s Shout:%s", yn(bc), yn(bo), yn(sh)))
`)
	if out != "BC:Y BO:Y Shout:Y" {
		t.Errorf("expected 'BC:Y BO:Y Shout:Y', got %q", out)
	}
}

// TestAndOrReturnValueDirectly tests that and/or returns the correct
// value type (string, not boolean) when used in return.
func TestAndOrReturnValueDirectly(t *testing.T) {
	results := runLuaReturning(t, `
local function yn(v) return v and "Y" or "N" end
return yn(true)
`)
	if len(results) == 0 {
		t.Fatal("expected a return value")
	}
	s := results[0].AsString()
	if s != "Y" {
		t.Errorf("expected 'Y', got %q (type issue: and/or returned raw boolean)", s)
	}
}

// TestAndOrChainPreservesType ensures the and/or chain returns the string
// "Y", not the boolean true, across multiple sequential calls.
func TestAndOrChainPreservesType(t *testing.T) {
	results := runLuaReturning(t, `
local function yn(v) return v and "Y" or "N" end
local r1 = yn(true)
local r2 = yn(true)
local r3 = yn(true)
return type(r1), type(r2), type(r3), r1, r2, r3
`)
	if len(results) < 6 {
		t.Fatalf("expected 6 return values, got %d", len(results))
	}
	for i := 0; i < 3; i++ {
		typ := results[i].AsString()
		if typ != "string" {
			t.Errorf("return %d: expected type 'string', got %q", i+1, typ)
		}
	}
	for i := 3; i < 6; i++ {
		val := results[i].AsString()
		if val != "Y" {
			t.Errorf("return %d: expected 'Y', got %q", i+1, val)
		}
	}
}

// TestMultiCallReturnFreeRegGap tests the same bug class in return statements:
// `return f(), g(), h()` where inner calls inflate freeReg, causing a gap
// in the return values when the last call is multi-ret.
func TestMultiCallReturnFreeRegGap(t *testing.T) {
	results := runLuaReturning(t, `
local function double(x) return x * 2 end
local function test()
    return double(10), double(20), double(30)
end
return test()
`)
	if len(results) < 3 {
		t.Fatalf("expected 3 return values, got %d", len(results))
	}
	expected := []int64{20, 40, 60}
	for i, exp := range expected {
		if results[i].AsInt() != exp {
			t.Errorf("return %d: expected %d, got %v", i+1, exp, results[i])
		}
	}
}

// TestMultiCallArgsFreeRegGapWithMultiArgFuncs tests with inner functions
// that take multiple arguments (more register pressure).
func TestMultiCallArgsFreeRegGapWithMultiArgFuncs(t *testing.T) {
	out := runLuaAndOr(t, `
local function add(a, b) return tostring(a + b) end
print(add(1, 2), add(3, 4), add(5, 6))
`)
	if out != "3\t7\t11" {
		t.Errorf("expected '3\\t7\\t11', got %q", out)
	}
}

// TestNestedCallsAsLastArg tests a function call whose argument is itself
// a function call, as the last argument to an outer call.
func TestNestedCallsAsLastArg(t *testing.T) {
	out := runLuaAndOr(t, `
local function double(x) return x * 2 end
local function inc(x) return x + 1 end
print(string.format("%d %d %d", double(1), double(2), inc(double(3))))
`)
	if out != "2 4 7" {
		t.Errorf("expected '2 4 7', got %q", out)
	}
}

// TestMethodCallArgsFreeRegGap tests the same bug class with method calls
// (obj:method(f(), g(), h())).
func TestMethodCallArgsFreeRegGap(t *testing.T) {
	out := runLuaAndOr(t, `
local function double(x) return x * 2 end
print(string.format("%d %d %d", double(5), double(10), double(15)))
`)
	if out != "10 20 30" {
		t.Errorf("expected '10 20 30', got %q", out)
	}
}

// TestFreeRegGapWithStringConcat tests that function call args don't create
// gaps even when the functions do string operations internally.
func TestFreeRegGapWithStringConcat(t *testing.T) {
	out := runLuaAndOr(t, `
local function wrap(s) return "[" .. s .. "]" end
print(wrap("a"), wrap("b"), wrap("c"))
`)
	if out != "[a]\t[b]\t[c]" {
		t.Errorf("expected '[a]\\t[b]\\t[c]', got %q", out)
	}
}
