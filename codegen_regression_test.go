package golua_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// runLuaCapture compiles and runs source with print() captured, returning the
// joined output lines. Used by the codegen regression tests to assert on
// computed values against reference Lua 5.5 behavior.
func runLuaCapture(t *testing.T, source string) string {
	t.Helper()
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New(vm.WithCaptureOutput(true))
	stdlib.Open(v)
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return strings.Join(v.OutputLines(), "\n")
}

// numeric-for per-iteration OP_CLOSE must not be skipped when the
// loop body declares an inlined <const> local. Closures must capture a fresh
// loop variable per iteration.
func TestForNumInlinedConstCapture(t *testing.T) {
	got := runLuaCapture(t, `
local CL = {}
for i = 1, 2 do
  local c <const> = 5
  CL[#CL+1] = function() return i + c end
end
print(CL[1](), CL[2]())`)
	if got != "6\t7" {
		t.Fatalf("closures shared loop var: got %q want %q", got, "6\t7")
	}
}

func TestForNumInlinedConstClose(t *testing.T) {
	got := runLuaCapture(t, `
for i = 1, 2 do
  local x <close> = setmetatable({}, {__close=function() print("close", i) end})
  local c <const> = 5
end`)
	if got != "close\t1\nclose\t2" {
		t.Fatalf("<close> fired with wrong per-iteration value: got %q", got)
	}
}

// call operands in concat/and/or chains and chained calls
// must reuse registers per operand/level instead of compounding, otherwise long
// (but valid) expressions spuriously overflow the register file.
func TestConcatChainRegisters(t *testing.T) {
	var b strings.Builder
	b.WriteString("local function f(n) return n end\nprint(")
	for i := 0; i < 60; i++ {
		if i > 0 {
			b.WriteString("..")
		}
		b.WriteString("f(1)")
	}
	b.WriteString(")")
	got := runLuaCapture(t, b.String())
	if got != strings.Repeat("1", 60) {
		t.Fatalf("concat chain miscompiled: got %q", got)
	}
}

func TestAndOrChainRegisters(t *testing.T) {
	var b strings.Builder
	b.WriteString("local function f(n) return n end\nprint(")
	for i := 0; i < 200; i++ {
		if i > 0 {
			b.WriteString(" and ")
		}
		b.WriteString("f(1)")
	}
	b.WriteString(")")
	if got := runLuaCapture(t, b.String()); got != "1" {
		t.Fatalf("and chain miscompiled: got %q", got)
	}
}

// a long non-constant left-associative arithmetic chain must
// compile in linear time (constant folding must not re-walk the left spine at
// every node). Guards against the O(n^2) compile-time DoS.
func TestCompileDoSLinear(t *testing.T) {
	build := func(n int) string {
		var b strings.Builder
		b.WriteString("local x=1 return ")
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString("+")
			}
			b.WriteString("x")
		}
		return b.String()
	}
	compile := func(src string) {
		block, err := parser.Parse("dos", src)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, err := compiler.Compile("dos", block); err != nil {
			t.Fatalf("compile: %v", err)
		}
	}
	// A large chain must compile quickly; a 15s budget is ~100x the observed
	// linear time and would blow out under the old quadratic behavior.
	done := make(chan struct{})
	go func() { compile(build(60000)); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("compiling a 60k-term non-constant chain took too long (quadratic?)")
	}
	// Constant chains must still fold to a single value.
	if got := runLuaCapture(t, "print("+strings.Repeat("1+", 4999)+"1)"); got != "5000" {
		t.Fatalf("constant fold broken: got %q want 5000", got)
	}
}

// a bare in-register local left operand must be read live at
// execution time, so a mutation the right operand makes to it (via a shared
// upvalue) is observed — matching reference Lua's exp2anyreg.
func TestBinaryLeftOperandLive(t *testing.T) {
	cases := []struct{ src, want string }{
		{`local u=1 local function bump() u=12; return 2 end print(u + bump())`, "14"},
		{`local u=10 local function bump() u=5; return 20 end print(u < bump())`, "true"},
		{`local a=3 local function b() a=100; return 1 end print(a > b())`, "true"},
		{`local u=10 local function m() u=1; return 2 end print(u <= m())`, "true"},
	}
	for _, c := range cases {
		if got := runLuaCapture(t, c.src); got != c.want {
			t.Fatalf("%s => got %q want %q", c.src, got, c.want)
		}
	}
}

func TestChainedCallRegisters(t *testing.T) {
	got := runLuaCapture(t, `
local function f() return f end
local g = f
for i = 1, 300 do g = g() end
print(g == f)`)
	if got != "true" {
		t.Fatalf("chained call miscompiled: got %q", got)
	}
	// Deep chained call must compile without a Go panic / register overflow.
	var b strings.Builder
	b.WriteString("local function f() return f end local g=f local x = g")
	for i := 0; i < 1000; i++ {
		b.WriteString("()")
	}
	b.WriteString(" print(x==f)")
	if got := runLuaCapture(t, b.String()); got != "true" {
		t.Fatalf("deep chained call miscompiled: got %q", got)
	}
}

// after a table constructor consumes an open call or varargs (OP_SETLIST with
// B=0), vm.top must be restored to the frame's register ceiling. A stale low
// vm.top let the next metamethod or string-coercion dispatch build its callee
// frame on top of live caller registers, silently corrupting them.
func TestSetListRestoresTop(t *testing.T) {
	cases := []struct{ src, want string }{
		{`local function mr() return 1, 2, 3 end
local obj = setmetatable({}, {__add = function() return 1000 end})
local t = {1, mr()}
print("a", "b", "c", "d", (obj + 1))`, "a\tb\tc\td\t1000"},
		{`local function g() return 1, 2 end
local function main(...)
  local t = {g()}
  local a, b, c = "A", "B", "C"
  local y = "21" % 3
  print(a, b, c, y)
end
main(1, "two", nil, 4)`, "A\tB\tC\t0"},
		{`local function main(...)
  local t = {...}
  local a = "A"
  local y = "21" % 3
  print(a, y)
end
main(1, 2, 3, 4)`, "A\t0"},
		{`local function mr() return 1, 2, 3 end
local obj = setmetatable({}, {__concat = function() return "CC" end})
local t = {1, mr()}
local u = {"a", "b", "c", "d", "e", (obj .. 1)}
print(u[4], u[5], u[6])`, "d\te\tCC"},
	}
	for _, c := range cases {
		if got := runLuaCapture(t, c.src); got != c.want {
			t.Fatalf("%s => got %q want %q", c.src, got, c.want)
		}
	}
}

// a forward goto to a body-end label (goto-continue) inside a for loop must
// not close the loop's own control locals: the generic-for closing value and
// captured loop variables stay live across a body-end label.
func TestGotoContinueForClose(t *testing.T) {
	cases := []struct{ src, want string }{
		{`local log, n = {}, 0
local tbc = setmetatable({}, {__close=function() log[#log+1]="CLOSE" end})
for i in function() n=n+1 if n<=2 then return n end end, nil, nil, tbc do
  log[#log+1] = "iter"..i
  goto cont
  ::cont::
end
print(table.concat(log, ","))`, "iter1,iter2,CLOSE"},
		{`local fs = {}
for i = 1, 3 do
  fs[i] = function() return i end
  if i == 2 then goto cont end
  ::cont::
end
print(fs[1](), fs[2](), fs[3]())`, "1\t2\t3"},
		{`local log2, m = {}, 0
for i in function() m=m+1 if m<=2 then return m end end do
  local b <close> = setmetatable({}, {__close=function() log2[#log2+1]="B"..i end})
  if i == 1 then goto cont2 end
  log2[#log2+1] = "work"..i
  ::cont2::
end
print(table.concat(log2, ","))`, "B1,work2,B2"},
	}
	for _, c := range cases {
		if got := runLuaCapture(t, c.src); got != c.want {
			t.Fatalf("%s => got %q want %q", c.src, got, c.want)
		}
	}
}

// nested table constructors must cost one register per level (checked): the
// old temp+MOVE path cost two and its manual freeReg bumps skipped the
// MaxRegs check, so >=129-deep constructors silently wrapped 8-bit register
// operands past 255 and built corrupt structures.
func TestDeepNestedConstructor(t *testing.T) {
	for _, depth := range []int{64, 129, 150, 190} {
		src := "local t = " + strings.Repeat("{", depth) + "42" + strings.Repeat("}", depth) +
			"\nlocal c, n = t, 0\nwhile type(c) == 'table' and c[1] ~= nil do c = c[1]; n = n + 1 end\nprint(n, c)"
		want := fmt.Sprintf("%d\t42", depth)
		if got := runLuaCapture(t, src); got != want {
			t.Fatalf("depth %d: got %q want %q", depth, got, want)
		}
		src = "local t = " + strings.Repeat("{k=", depth) + "1" + strings.Repeat("}", depth) +
			"\nlocal c, n = t, 0\nwhile type(c) == 'table' and c.k ~= nil do c = c.k; n = n + 1 end\nprint(n, c)"
		want = fmt.Sprintf("%d\t1", depth)
		if got := runLuaCapture(t, src); got != want {
			t.Fatalf("hash depth %d: got %q want %q", depth, got, want)
		}
	}
}
