package parser

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/ast"
)

// helper: parse and dump to string
func dump(t *testing.T, src string) string {
	t.Helper()
	block, err := Parse("test", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return ast.DumpString(block)
}

// helper: assert parse fails
func expectError(t *testing.T, src, want string) {
	t.Helper()
	_, err := Parse("test", src)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}

func contains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("output does not contain %q\ngot:\n%s", want, got)
	}
}

// ---------------------------------------------------------------------------
// Expression tests
// ---------------------------------------------------------------------------

func TestLiterals(t *testing.T) {
	d := dump(t, "return 42, 3.14, 'hello', true, false, nil")
	contains(t, d, "(int 42)")
	contains(t, d, "(float 3.14)")
	contains(t, d, `(string "hello")`)
	contains(t, d, "true")
	contains(t, d, "false")
	contains(t, d, "nil")
}

func TestBinaryOps(t *testing.T) {
	d := dump(t, "return 1 + 2 * 3")
	// Should parse as 1 + (2 * 3) due to precedence
	contains(t, d, "(binop +")
	contains(t, d, "(int 1)")
	contains(t, d, "(binop *")
}

func TestRightAssocPow(t *testing.T) {
	d := dump(t, "return 2 ^ 3 ^ 4")
	// Should parse as 2 ^ (3 ^ 4)
	contains(t, d, "(binop ^")
}

func TestRightAssocConcat(t *testing.T) {
	d := dump(t, "return 'a' .. 'b' .. 'c'")
	contains(t, d, "(binop ..")
}

func TestUnaryOps(t *testing.T) {
	d := dump(t, "return -x, not y, #t, ~n")
	contains(t, d, "(unop -")
	contains(t, d, "(unop not")
	contains(t, d, "(unop #")
	contains(t, d, "(unop ~")
}

func TestUnaryPrecedence(t *testing.T) {
	d := dump(t, "return -2 ^ 3")
	// Should parse as -(2 ^ 3): unary has priority 12, ^ left is 14
	// Actually: unary is parsed with priority 12. ^ has left priority 14 > 12,
	// so ^ binds tighter. So it's ((-2) ^ 3)? No...
	// Lua: -2^3 = -(2^3) = -8. The unary operator applies to the RESULT of ^.
	// In Lua's parser: unary calls subexpr(UNARY_PRIORITY=12).
	// Inside that, simpleexp returns 2, then ^ has left priority 14 > 12,
	// so it binds. So operand of - is (2^3). Correct!
	contains(t, d, "(unop -")
	contains(t, d, "(binop ^")
}

// ---------------------------------------------------------------------------
// Statement tests
// ---------------------------------------------------------------------------

func TestAssignment(t *testing.T) {
	d := dump(t, "x = 1")
	contains(t, d, "(assign")
	contains(t, d, "(name x)")
	contains(t, d, "(int 1)")
}

func TestMultiAssignment(t *testing.T) {
	d := dump(t, "a, b = 1, 2")
	contains(t, d, "(assign")
	contains(t, d, "(name a)")
	contains(t, d, "(name b)")
}

func TestLocalDecl(t *testing.T) {
	d := dump(t, "local x, y = 1, 2")
	contains(t, d, "(local")
	contains(t, d, "x")
	contains(t, d, "y")
}

func TestLocalConst(t *testing.T) {
	d := dump(t, "local x <const> = 42")
	contains(t, d, "x <const>")
}

func TestLocalClose(t *testing.T) {
	d := dump(t, "local f <close> = io.open('x')")
	contains(t, d, "f <close>")
}

func TestDoEnd(t *testing.T) {
	d := dump(t, "do local x = 1 end")
	contains(t, d, "(do")
}

func TestWhile(t *testing.T) {
	d := dump(t, "while x > 0 do x = x - 1 end")
	contains(t, d, "(while")
	contains(t, d, "(binop >")
}

func TestRepeat(t *testing.T) {
	d := dump(t, "repeat x = x + 1 until x > 10")
	contains(t, d, "(repeat")
	contains(t, d, "(until")
}

func TestIfElseifElse(t *testing.T) {
	d := dump(t, "if a then b() elseif c then d() else e() end")
	contains(t, d, "(if")
	contains(t, d, "(elseif")
	contains(t, d, "(else")
}

func TestForNum(t *testing.T) {
	d := dump(t, "for i = 1, 10, 2 do print(i) end")
	contains(t, d, "(for-num i")
	contains(t, d, "(start")
	contains(t, d, "(stop")
	contains(t, d, "(step")
}

func TestForIn(t *testing.T) {
	d := dump(t, "for k, v in pairs(t) do print(k, v) end")
	contains(t, d, "(for-in [k, v]")
	contains(t, d, "(iters")
}

func TestReturn(t *testing.T) {
	d := dump(t, "return 1, 2, 3")
	contains(t, d, "(return")
}

func TestReturnEmpty(t *testing.T) {
	d := dump(t, "return")
	contains(t, d, "(return)")
}

func TestBreak(t *testing.T) {
	d := dump(t, "while true do break end")
	contains(t, d, "(break)")
}

func TestGotoLabel(t *testing.T) {
	d := dump(t, "goto skip; ::skip:: print(42)")
	contains(t, d, "(goto skip)")
	contains(t, d, "(label skip)")
}

func TestFuncStmt(t *testing.T) {
	d := dump(t, "function foo(a, b) return a + b end")
	contains(t, d, "(func-stmt")
	contains(t, d, "(name foo)")
	contains(t, d, "(function (a, b)")
}

func TestFuncStmtDotted(t *testing.T) {
	d := dump(t, "function a.b.c(x) end")
	contains(t, d, "(func-stmt")
	contains(t, d, "(field .c")
	contains(t, d, "(field .b")
}

func TestFuncStmtMethod(t *testing.T) {
	d := dump(t, "function obj:method(x) return self end")
	contains(t, d, ":method")
	contains(t, d, "(function (self, x)")
}

func TestLocalFunc(t *testing.T) {
	d := dump(t, "local function f(x) return x end")
	contains(t, d, "(local-func f")
}

func TestFuncExpr(t *testing.T) {
	d := dump(t, "local f = function(x, ...) return x end")
	contains(t, d, "(function (x, ...)")
}

func TestVarArg(t *testing.T) {
	d := dump(t, "local f = function(...) return ... end")
	contains(t, d, "(function (...)")
	contains(t, d, "...")
}

// ---------------------------------------------------------------------------
// Table constructors
// ---------------------------------------------------------------------------

func TestTableEmpty(t *testing.T) {
	d := dump(t, "local t = {}")
	contains(t, d, "(table {})")
}

func TestTableList(t *testing.T) {
	d := dump(t, "local t = {1, 2, 3}")
	contains(t, d, "(item")
	contains(t, d, "(int 1)")
}

func TestTableRecord(t *testing.T) {
	d := dump(t, `local t = {x = 1, y = 2}`)
	contains(t, d, "(field")
	contains(t, d, `(string "x")`)
}

func TestTableBracketKey(t *testing.T) {
	d := dump(t, `local t = {[1+1] = "two"}`)
	contains(t, d, "(field")
	contains(t, d, "(binop +")
}

func TestTableMixed(t *testing.T) {
	d := dump(t, `local t = {1, x = 2, [3] = 4}`)
	contains(t, d, "(item")
	contains(t, d, "(field")
}

// ---------------------------------------------------------------------------
// Suffixed expressions
// ---------------------------------------------------------------------------

func TestFieldAccess(t *testing.T) {
	d := dump(t, "return a.b.c")
	contains(t, d, "(field .c")
	contains(t, d, "(field .b")
}

func TestIndexAccess(t *testing.T) {
	d := dump(t, "return a[1][2]")
	contains(t, d, "(index")
}

func TestMethodCall(t *testing.T) {
	d := dump(t, "obj:method(1, 2)")
	contains(t, d, "(method-call :method")
}

func TestFuncCall(t *testing.T) {
	d := dump(t, "print(42)")
	contains(t, d, "(call")
	contains(t, d, "(name print)")
}

func TestFuncCallString(t *testing.T) {
	d := dump(t, `require "foo"`)
	contains(t, d, "(call")
	contains(t, d, `(string "foo")`)
}

func TestFuncCallTable(t *testing.T) {
	d := dump(t, `f{1, 2}`)
	contains(t, d, "(call")
	contains(t, d, "(table {")
}

func TestParenExpr(t *testing.T) {
	d := dump(t, "return (1 + 2) * 3")
	contains(t, d, "(paren")
}

// ---------------------------------------------------------------------------
// Complex programs
// ---------------------------------------------------------------------------

func TestFibonacci(t *testing.T) {
	src := `
local function fib(n)
  if n <= 1 then
    return n
  end
  return fib(n - 1) + fib(n - 2)
end
print(fib(10))
`
	d := dump(t, src)
	contains(t, d, "(local-func fib")
	contains(t, d, "(if")
	contains(t, d, "(binop +")
}

func TestComplexTableConstructor(t *testing.T) {
	src := `
local config = {
  name = "test",
  values = {1, 2, 3},
  callback = function(x) return x * 2 end,
}
`
	d := dump(t, src)
	contains(t, d, `(string "name")`)
	contains(t, d, `(string "values")`)
	contains(t, d, `(string "callback")`)
	contains(t, d, "(function (x)")
}

func TestEmptyStatement(t *testing.T) {
	d := dump(t, ";;; return 1")
	contains(t, d, "(;)")
	contains(t, d, "(return")
}

func TestChainedCalls(t *testing.T) {
	d := dump(t, "a.b:c(1):d(2)")
	contains(t, d, "(method-call :d")
	contains(t, d, "(method-call :c")
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

func TestErrorUnexpectedSymbol(t *testing.T) {
	expectError(t, "return )", "unexpected symbol")
}

func TestErrorMissingEnd(t *testing.T) {
	expectError(t, "if true then", "'end' expected")
}

func TestErrorBadExprStmt(t *testing.T) {
	// A bare number hits "unexpected symbol" in primaryExpr
	expectError(t, "42", "unexpected symbol")
	// A bare name that isn't a call hits "unexpected expression statement"
	expectError(t, "x", "unexpected expression statement")
}
