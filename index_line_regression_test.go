package golua_test

import (
	"strings"
	"testing"
)

// Reference Lua stamps an instruction with the line of the last token consumed
// before it is emitted, and it reads the whole suffix — the field name, the
// closing ']' of an index, the method name — before the pending index is
// discharged. An error in a suffixed expression split over several lines
// therefore reports the suffix's line, not the line of the '.', '[' or ':'
// that introduced it.
func TestSuffixedExprErrorLine(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		wantLine  string
	}{
		{"field", "local a = nil\nreturn a\n.\nx\n", "]:4:"},
		{"index", "local a = nil\nreturn a[\n1\n]\n", "]:4:"},
		{"method", "local s = 'x'\nreturn s\n:\nrep(nil)\n", "]:4:"},
		{"chained field", "local t = {}\nreturn t\n  .a\n  .b\n", "]:4:"},
	} {
		got := runLuaExpectError(t, tc.src)
		if !strings.Contains(got, tc.wantLine) {
			t.Errorf("%s: got %q, want line %s", tc.name, got, tc.wantLine)
		}
	}
}

// A record field's store instruction is emitted after the field's value has
// been read, so it carries that value's line rather than the line of the '{'
// that opened the constructor.
func TestTableConstructorFieldErrorLine(t *testing.T) {
	got := runLuaExpectError(t, "local a = nil\nreturn {\n[\na\n]\n=\n1\n}\n")
	if !strings.Contains(got, "]:7:") {
		t.Errorf("bracketed key: got %q, want line 7", got)
	}

	got = runLuaExpectError(t, "local k = nil\nlocal t = {\n  1,\n  2,\n  3,\n  [k] = 4,\n}\nreturn t\n")
	if !strings.Contains(got, "]:6:") {
		t.Errorf("key several lines below '{': got %q, want line 6", got)
	}
}
