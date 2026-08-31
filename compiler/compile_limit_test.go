package compiler

import (
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/ast"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/token"
)

// TestTooManyLocalsStopsAtLimit is a regression guard for the compile-time
// O(n^2) DoS where, after recording a "too many local variables" error, the
// compiler kept walking every remaining statement (each doing O(active-locals)
// work) instead of stopping like reference Lua's longjmp-on-error. A chunk with
// 200k local declarations took ~18s (O(n^2)) before the fix and ~0.1s after
// (it stops at the 201st local). The error itself is unchanged.
func TestTooManyLocalsStopsAtLimit(t *testing.T) {
	const n = 200000
	var sb strings.Builder
	sb.Grow(n * 16)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "local _x%d = %d\n", i, i)
	}
	sb.WriteString("return 1\n")

	block, err := parser.Parse("<test>", sb.String())
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	start := time.Now()
	_, cerr := Compile("<test>", block)
	elapsed := time.Since(start)

	if cerr == nil {
		t.Fatal("expected a 'too many local variables' compile error")
	}
	if !strings.Contains(cerr.Error(), "too many local variables (limit is 200)") {
		t.Fatalf("unexpected error: %v", cerr)
	}
	// Generous bound: post-fix ~0.1s, pre-fix (O(n^2)) many seconds.
	if elapsed > 5*time.Second {
		t.Fatalf("compiling %d locals took %v — O(n^2) regression (compiler must stop at first error)", n, elapsed)
	}
}

// TestDeepIndexChainReusesRegister is a regression guard for chained
// index/field expressions (t.a.b.c... / t[1][1][1]...) reserving a fresh
// register per level. That overflowed the 255-register limit at chain depth
// ~255 on programs reference Lua compiles fine; the compiler must reuse one
// register down the chain (GETFIELD reg, reg, k).
func TestDeepIndexChainReusesRegister(t *testing.T) {
	for _, tc := range []struct {
		name, src string
	}{
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
