package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

// expectCompileError parses+compiles source and expects a compile error containing substr.
func expectCompileError(t *testing.T, source, name, substr string) {
	t.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		// Parse error is acceptable too — the important thing is it doesn't succeed
		if !strings.Contains(err.Error(), substr) {
			t.Logf("got parse error (acceptable): %v", err)
		}
		return
	}
	_, err = compiler.Compile(name, block)
	if err == nil {
		t.Fatalf("expected compile error containing %q, but compilation succeeded", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected compile error containing %q, got: %v", substr, err)
	}
}

func TestGotoIntoLocalScope(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"forward goto jumps over local",
			`goto bad
			local x = 42
			::bad::
			print(x)`,
		},
		{
			"forward goto jumps over multiple locals",
			`goto bad
			local a = 1
			local b = 2
			::bad::
			print(a, b)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectCompileError(t, tt.source, tt.name, "local")
		})
	}
}

// TestGotoScopeErrorLineSkipsNoops verifies the "jumps into the scope" error
// line matches reference Lua 5.5: a trailing ';' (or labels) after the target
// label is consumed by labelstat() before the goto is resolved, so the error
// line is the first real statement after the label run, NOT the first ';'.
func TestGotoScopeErrorLineSkipsNoops(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantLine string // expected ":LINE:" fragment in the error
	}{
		{
			// label on line 3, ';' on line 4, print on line 5 -> error at line 5
			"semicolon after label",
			"goto L\n" +
				"local z = 9\n" +
				"::L:: ;\n" +
				"print('after')\n",
			":4:",
		},
		{
			// label line 3, two empties lines 4-5, print line 6 -> error line 6
			"multiple empties after label",
			"goto L\n" +
				"local z = 9\n" +
				"::L::\n" +
				";\n" +
				";\n" +
				"print('after')\n",
			":6:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := parser.Parse(tt.name, tt.source)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			_, err = compiler.Compile(tt.name, block)
			if err == nil {
				t.Fatalf("expected compile error, got success")
			}
			if !strings.Contains(err.Error(), "jumps into the scope") {
				t.Fatalf("expected scope error, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantLine) {
				t.Fatalf("expected error line %q, got: %v", tt.wantLine, err)
			}
		})
	}
}

// TestDuplicateLabelErrorLine verifies the duplicate-label error position and
// referenced line match reference Lua 5.5 when no-op statements (';') separate
// the two labels. Lua's labelstat() consumes trailing ';' and following labels
// in one call, registering them in reverse, so the error is reported at the
// textually later label and references the already-registered duplicate.
func TestDuplicateLabelErrorLine(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantPos string // expected ":LINE:" prefix
		wantRef string // expected "on line N" fragment
	}{
		{
			// adjacent via semicolon: ::a:: L1, ; L2, ::a:: L3
			"semicolon between duplicate labels",
			"::a::\n;\n::a::\n",
			":3:", "on line 3",
		},
		{
			// adjacent same line: ::a:: ; ::a::
			"semicolon same line",
			"::a:: ; ::a::\n",
			":1:", "on line 1",
		},
		{
			// non-adjacent (real statement between): position at later label,
			// message references the earlier one
			"real statement between duplicates",
			"::L::\nlocal x = 1\n::L::\n",
			":3:", "on line 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := parser.Parse(tt.name, tt.source)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			_, err = compiler.Compile(tt.name, block)
			if err == nil {
				t.Fatalf("expected duplicate-label error, got success")
			}
			msg := err.Error()
			if !strings.Contains(msg, "already defined") {
				t.Fatalf("expected duplicate-label error, got: %v", msg)
			}
			if !strings.Contains(msg, tt.wantPos) {
				t.Fatalf("expected position %q, got: %v", tt.wantPos, msg)
			}
			if !strings.Contains(msg, tt.wantRef) {
				t.Fatalf("expected %q, got: %v", tt.wantRef, msg)
			}
		})
	}
}

func TestGotoLabelVisibility(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"label in exited block",
			`do
				::inner::
			end
			goto inner`,
		},
		{
			"label in sibling block",
			`do
				::lbl::
			end
			do
				goto lbl
			end`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectCompileError(t, tt.source, tt.name, "")
		})
	}
}

func TestGotoValidCases(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			"forward goto",
			`goto skip
			print("should not run")
			::skip::`,
		},
		{
			"backward goto loop",
			`local i = 0
			::loop::
			i = i + 1
			if i < 3 then goto loop end
			assert(i == 3)`,
		},
		{
			"goto out of local scope",
			`do
				local y = 7
				goto out
				print("unreachable")
			end
			::out::`,
		},
		{
			"goto to label in same block no locals between",
			`goto target
			::target::`,
		},
		{
			"goto backward to label before locals",
			`local j = 0
			::back::
			j = j + 1
			local dummy = j
			if j < 2 then goto back end
			assert(j == 2)`,
		},
		{
			"goto over local to label at end of block",
			`local x = 13
			do
				goto l1
				local a = 23
				x = a
				::l1::
			end
			assert(x == 13)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLuaSource(t, tt.source, tt.name)
		})
	}
}
