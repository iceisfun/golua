package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

// compileErr parses+compiles and returns the compile error string (or "").
func compileErr(t *testing.T, source, name string) string {
	t.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		return err.Error()
	}
	if _, err := compiler.Compile(name, block); err != nil {
		return err.Error()
	}
	return ""
}

// TestTooManyRegisters_Wording verifies the Lua 5.5 register-limit message,
// reworded from the 5.4 "function or expression needs too many registers" to
// "too many registers (limit is N) in main function/in function at line N
// near '<token>'".
func TestTooManyRegisters_Wording(t *testing.T) {
	manyArgs := "x" + strings.Repeat(",x", 260)

	// In the main chunk.
	got := compileErr(t, "local x=1; a = f("+manyArgs+")", "chunk")
	want := "too many registers (limit is 255) in main function near 'x'"
	if !strings.Contains(got, want) {
		t.Fatalf("main chunk: got %q, want substring %q", got, want)
	}

	// Inside a function: scope clause becomes "in function at line N".
	got = compileErr(t, "local x=1\nfunction g() return f("+manyArgs+") end", "chunk")
	want = "too many registers (limit is 255) in function at line 2 near 'x'"
	if !strings.Contains(got, want) {
		t.Fatalf("nested function: got %q, want substring %q", got, want)
	}
}
