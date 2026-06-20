package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/parser"
)

// TestCStackOverflow_NoPositionPrefix verifies that the parser's depth-limit
// error is the bare string "C stack overflow", with no "source:line:" prefix.
// Reference Lua raises this via luaD_throw rather than luaX_syntaxerror, so it
// carries no position (unlike ordinary syntax errors).
func TestCStackOverflow_NoPositionPrefix(t *testing.T) {
	deep := "return " + strings.Repeat("(", 400) + "1" + strings.Repeat(")", 400)
	_, err := parser.Parse("chunk", deep)
	if err == nil {
		t.Fatal("expected a depth-limit error")
	}
	if got := err.Error(); got != "C stack overflow" {
		t.Fatalf("got %q, want bare %q (no position prefix)", got, "C stack overflow")
	}
}
