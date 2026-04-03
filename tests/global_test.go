package tests

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

// mustCompile parses and compiles code, returning any error.
func mustCompileGlobal(code string) error {
	block, err := parser.Parse("test", code)
	if err != nil {
		return err
	}
	_, err = compiler.Compile("test", block)
	return err
}

func TestGlobalDeclBasic(t *testing.T) {
	// Declared global can be assigned and read
	err := mustCompileGlobal(`global X; global print; X = 1; print(X)`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclUndeclaredRead(t *testing.T) {
	err := mustCompileGlobal(`global X; global print; print(Y)`)
	if err == nil {
		t.Fatal("expected compile error for undeclared read")
	}
	if !strings.Contains(err.Error(), "variable 'Y' not declared") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclUndeclaredWrite(t *testing.T) {
	err := mustCompileGlobal(`global X; Y = 1`)
	if err == nil {
		t.Fatal("expected compile error for undeclared write")
	}
	if !strings.Contains(err.Error(), "variable 'Y' not declared") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclConstWrite(t *testing.T) {
	err := mustCompileGlobal(`global<const> X; X = 1`)
	if err == nil {
		t.Fatal("expected compile error for const write")
	}
	if !strings.Contains(err.Error(), "attempt to assign to const variable 'X'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclConstRead(t *testing.T) {
	err := mustCompileGlobal(`global<const> print; print("hello")`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclWildcard(t *testing.T) {
	err := mustCompileGlobal(`global print; global *; Y = 1; print(Y)`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclConstWildcard(t *testing.T) {
	err := mustCompileGlobal(`global<const> *; X = 1`)
	if err == nil {
		t.Fatal("expected compile error for const wildcard write")
	}
	if !strings.Contains(err.Error(), "attempt to assign to const variable 'X'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclConstWildcardRead(t *testing.T) {
	err := mustCompileGlobal(`global<const> *; print("hi")`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclSpecificOverridesConstWildcard(t *testing.T) {
	err := mustCompileGlobal(`global print; global<const> *; print = nil`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclNestedScopeInherits(t *testing.T) {
	err := mustCompileGlobal(`global X; do X = 1 end`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclNestedScopeOwnDecls(t *testing.T) {
	err := mustCompileGlobal(`global X; do global Y; Y = 1; X = 1 end`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclInnerScopeVoidsImplicit(t *testing.T) {
	err := mustCompileGlobal(`do global Y; X = 1 end`)
	if err == nil {
		t.Fatal("expected compile error for undeclared X in inner scope")
	}
	if !strings.Contains(err.Error(), "variable 'X' not declared") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclFunctionBodyInherits(t *testing.T) {
	// Inner function body inherits outer scope's global restrictions
	err := mustCompileGlobal(`global X; local f = function() Y = 1 end`)
	if err == nil {
		t.Fatal("expected compile error for undeclared Y in function body")
	}
	if !strings.Contains(err.Error(), "variable 'Y' not declared") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclFunctionBodyDeclared(t *testing.T) {
	// Inner function can use names declared in outer scope
	err := mustCompileGlobal(`global X; local f = function() return X end`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclFunctionBodyOwnGlobal(t *testing.T) {
	// Inner function can add its own global declarations
	err := mustCompileGlobal(`global X; local f = function() global Y; Y = 1 end`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclFunction(t *testing.T) {
	err := mustCompileGlobal(`global function f() return 42 end; f()`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclCumulative(t *testing.T) {
	err := mustCompileGlobal(`global X; global Y; X = 1; Y = 1`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclScopeRestore(t *testing.T) {
	// After leaving inner scope with explicit decls, outer scope rules apply
	err := mustCompileGlobal(`
		global *
		do
			global X
		end
		Z = 99
	`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestGlobalDeclFuncStmtWrite(t *testing.T) {
	// function foo() end is a write to _ENV["foo"]
	err := mustCompileGlobal(`global X; function foo() end`)
	if err == nil {
		t.Fatal("expected compile error for undeclared function assignment")
	}
	if !strings.Contains(err.Error(), "variable 'foo' not declared") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclConstFuncStmtWrite(t *testing.T) {
	// Cannot write to a const-declared global via function statement
	err := mustCompileGlobal(`global<const> foo; function foo() end`)
	if err == nil {
		t.Fatal("expected compile error for const function assignment")
	}
	if !strings.Contains(err.Error(), "attempt to assign to const variable 'foo'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalDeclMultiAssign(t *testing.T) {
	// Multi-assignment: a, b = 1, 2 uses assignToTarget path
	err := mustCompileGlobal(`global X; X, Y = 1, 2`)
	if err == nil {
		t.Fatal("expected compile error for undeclared Y in multi-assign")
	}
	if !strings.Contains(err.Error(), "variable 'Y' not declared") {
		t.Fatalf("unexpected error: %v", err)
	}
}
