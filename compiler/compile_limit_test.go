package compiler

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/parser"
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
