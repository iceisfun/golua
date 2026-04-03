package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// compileGlobalTest is a test helper that parses and compiles Lua source.
func compileGlobalTest(t *testing.T, source string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

// runGlobalTest runs the given Lua source and returns the result and error.
func runGlobalTest(t *testing.T, source string) ([]vm.Value, error) {
	t.Helper()
	proto := compileGlobalTest(t, source)
	v := vm.New()
	stdlib.Open(v)
	return v.Run(proto)
}

func TestGlobalInitCheck_SimpleInit(t *testing.T) {
	_, err := runGlobalTest(t, `
		global X = 10
		global assert
		assert(X == 10)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalInitCheck_AlreadyDefined(t *testing.T) {
	_, err := runGlobalTest(t, `
		X = 5
		global X = 10
	`)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
	if !strings.Contains(err.Error(), "global 'X' already defined") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGlobalInitCheck_DoubleInit(t *testing.T) {
	_, err := runGlobalTest(t, `
		global X = 10
		global X = 20
	`)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
	if !strings.Contains(err.Error(), "global 'X' already defined") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGlobalInitCheck_DeclarationNoInit(t *testing.T) {
	_, err := runGlobalTest(t, `
		X = 42
		global X, assert
		assert(X == 42)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalInitCheck_DeclarationThenAssign(t *testing.T) {
	_, err := runGlobalTest(t, `
		global Z, assert
		Z = 99
		assert(Z == 99)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalInitCheck_PcallCatchesError(t *testing.T) {
	_, err := runGlobalTest(t, `
		global *
		X = 5
		local f = load("global X = 10")
		local ok, msg = pcall(f)
		assert(ok == false, "expected pcall to return false")
		assert(type(msg) == "string", "expected error to be a string")
		assert(string.find(msg, "global 'X' already defined"),
			"expected error about X already defined, got: " .. msg)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalInitCheck_GlobalFunction(t *testing.T) {
	_, err := runGlobalTest(t, `
		foo = 42
		global function foo() end
	`)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
	if !strings.Contains(err.Error(), "global 'foo' already defined") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGlobalInitCheck_GlobalFunctionDouble(t *testing.T) {
	_, err := runGlobalTest(t, `
		global function bar() return 1 end
		global function bar() return 2 end
	`)
	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}
	if !strings.Contains(err.Error(), "global 'bar' already defined") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGlobalInitCheck_NilValueOK(t *testing.T) {
	_, err := runGlobalTest(t, `
		global X, assert
		X = nil
		global X = 10
		assert(X == 10)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobalInitCheck_MultipleNames(t *testing.T) {
	_, err := runGlobalTest(t, `
		global X, Y, assert = 1, 2
		assert(X == 1)
		assert(Y == 2)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = runGlobalTest(t, `
		Y = 99
		global X, Y = 1, 2
	`)
	if err == nil {
		t.Fatal("expected runtime error for Y, got nil")
	}
	if !strings.Contains(err.Error(), "global 'Y' already defined") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGlobalInitCheck_ErrorMessage(t *testing.T) {
	_, err := runGlobalTest(t, `
		X = 10
		global X = 20
	`)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "global 'X' already defined") {
		t.Fatalf("error message mismatch: %q", msg)
	}
}
