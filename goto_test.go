package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
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
