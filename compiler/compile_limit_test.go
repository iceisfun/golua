package compiler

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/iceisfun/golua/ast"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/token"
)

// TestDeepIndexChainReusesRegister guards chained index/field and and/or
// expressions against reserving a fresh register per level (which overflowed
// the 255-register limit at depth ~255 on programs reference Lua compiles).
// (The master-branch O(n^2) too-many-locals compile DoS does not apply here:
// 5.4.8 enforces the local limit in the parser, not the compiler.)
func TestDeepIndexChainReusesRegister(t *testing.T) {
	for _, tc := range []struct{ name, src string }{
		{"field", "local t = {}\nreturn t" + strings.Repeat(".f", 500)},
		{"index", "local t = {}\nreturn t" + strings.Repeat("[1]", 500)},
		{"and", "return " + strings.Repeat("1 and ", 500) + "1"},
		{"or", "return " + strings.Repeat("nil or ", 500) + "1"},
	} {
		block, err := parser.Parse("<test>", tc.src)
		if err != nil {
			t.Fatalf("%s: parse error: %v", tc.name, err)
		}
		if _, cerr := Compile("<test>", block); cerr != nil {
			t.Fatalf("%s: chain depth 500 should compile (reference does), got: %v", tc.name, cerr)
		}
	}
}

// TestLongSpineCompilesInBoundedStack guards the iterative compilation of
// left-leaning chains. The parser builds `a+a+a...`, `t.a.b.c...`, `f()()...`
// and `x and y and z...` iteratively, so a machine-generated chunk can hold
// millions of links; compiling one with a Go frame per link exhausted the
// goroutine stack, which is a fatal, uncatchable crash on source that reference
// Lua compiles in constant stack. The goroutine stack is capped for the
// duration so a regression trips on a short chain instead of needing a gigabyte
// of stack to show up.
func TestLongSpineCompilesInBoundedStack(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	const n = 60000
	for _, tc := range []struct{ name, src string }{
		{"add", "local a = 1\nreturn a" + strings.Repeat("+a", n)},
		{"mixed-arith", "local a = 1\nreturn a" + strings.Repeat("+a*a-a", n/3)},
		{"and", "local a = 1\nreturn a" + strings.Repeat(" and a", n)},
		{"or", "local a = 1\nreturn a" + strings.Repeat(" or a", n)},
		{"local-target", "local a, b = 1, 2\nb = b" + strings.Repeat("+a", n) + "\nreturn b"},
		{"field", "local t = {}\nreturn t" + strings.Repeat(".f", n)},
		{"index", "local t = {}\nreturn t" + strings.Repeat("[1]", n)},
		{"call", "local f = print\nreturn f" + strings.Repeat("()", n)},
		{"method", "local t = {}\nreturn t" + strings.Repeat(":m()", n)},
		{"call-field", "local t = {}\nreturn t" + strings.Repeat(".f(1)", n)},
	} {
		block, err := parser.Parse("<test>", tc.src)
		if err != nil {
			t.Fatalf("%s: parse error: %v", tc.name, err)
		}
		proto, cerr := Compile("<test>", block)
		if cerr != nil {
			t.Fatalf("%s: a chain of %d links should compile (reference does), got: %v", tc.name, n, cerr)
		}
		if len(proto.Code) == 0 {
			t.Fatalf("%s: compiled to no code", tc.name)
		}
	}
}

// TestConstantFoldingWalksLongChainIteratively covers the same shape on the
// constant-folding path, which walks the left spine of a chain before any code
// is emitted.
func TestConstantFoldingWalksLongChainIteratively(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	const n = 200000
	block, err := parser.Parse("<test>", "return 1"+strings.Repeat("+1", n))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, cerr := Compile("<test>", block)
	if cerr != nil {
		t.Fatalf("a folded chain of %d links should compile, got: %v", n, cerr)
	}
	// The whole chain folds to one constant load plus the return.
	if len(proto.Code) > 4 {
		t.Fatalf("expected the chain to fold to a single load, got %d instructions", len(proto.Code))
	}
}

// TestDeeplyNestedExprIsAnError covers the depth limit on nested expression
// compilation. The parser bounds its own recursion well below that limit, so it
// is reachable only through a syntax tree built directly — by an embedder
// generating code, or a source-to-source tool. Nesting deep enough to exhaust
// the Go stack must come back as an ordinary compile error, in the bare form
// the parser uses for its own limit, rather than ending the process.
func TestDeeplyNestedExprIsAnError(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	pos := token.Pos{Source: "<test>", Line: 1, Column: 1}
	var expr ast.Expr = ast.NewNumberExpr(pos, 1, "1")
	for i := 0; i < maxExprDepth*2; i++ {
		expr = ast.NewParenExpr(pos, expr)
	}
	block := &ast.Block{Start: pos, EndLine: 1, Stmts: []ast.Stmt{ast.NewReturnStmt(pos, []ast.Expr{expr})}}

	_, err := Compile("<test>", block)
	if err == nil {
		t.Fatal("expected a depth-limit error")
	}
	if got := err.Error(); got != "C stack overflow" {
		t.Fatalf("got %q, want bare %q (no position prefix)", got, "C stack overflow")
	}
}
