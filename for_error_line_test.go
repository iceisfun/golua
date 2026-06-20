package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/vm"
)

// runtimeErr compiles+runs source and returns the runtime error string.
func runtimeErr(t *testing.T, source, name string) string {
	t.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	if _, err := v.Run(proto); err != nil {
		return err.Error()
	}
	t.Fatalf("expected a runtime error, got none")
	return ""
}

// TestForNum_ErrorLine_AtDoKeyword verifies that numeric-for prep errors
// (bad initial value/limit/step, and "'for' step is zero") are attributed to
// the line of the 'do' keyword, matching reference Lua (which stamps the
// FORPREP instruction with lastline after consuming 'do'), not the 'for'
// keyword's line.
func TestForNum_ErrorLine_AtDoKeyword(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		wantSub string
	}{
		{"limit on later line", "for i=1,\nnil do end", "]:2: bad 'for' limit"},
		{"init on later line", "for i=\nnil,10 do end", "]:2: bad 'for' initial value"},
		{"step on later line", "for i=1,10,\nnil do end", "]:2: bad 'for' step"},
		{"do on its own line", "for\ni=1,nil\ndo end", "]:3: bad 'for' limit"},
		{"step zero at do line", "for i=1,10,0\ndo end", "]:2: 'for' step is zero"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runtimeErr(t, tc.source, "chunk")
			if !strings.Contains(got, tc.wantSub) {
				t.Fatalf("got %q, want substring %q", got, tc.wantSub)
			}
		})
	}
}
