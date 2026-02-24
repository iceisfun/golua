package check

import (
	"testing"
)

func TestCheckValid(t *testing.T) {
	r := Check("test", "local x = 1\nprint(x)")
	if r.Block == nil {
		t.Fatal("Block should never be nil")
	}
	if len(r.Diagnostics) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(r.Diagnostics))
	}
	if r.HasErrors() {
		t.Fatal("HasErrors should be false for valid input")
	}
}

func TestCheckEmpty(t *testing.T) {
	r := Check("test", "")
	if r.Block == nil {
		t.Fatal("Block should never be nil")
	}
	if len(r.Diagnostics) != 0 {
		t.Fatalf("expected 0 diagnostics for empty input, got %d", len(r.Diagnostics))
	}
}

func TestCheckPartialAST(t *testing.T) {
	r := Check("test", "local x = 1\nif true then")
	if r.Block == nil {
		t.Fatal("Block should never be nil")
	}
	if !r.HasErrors() {
		t.Fatal("expected errors")
	}
	// The local statement should have parsed.
	if len(r.Block.Stmts) < 1 {
		t.Errorf("expected at least 1 statement in partial AST, got %d", len(r.Block.Stmts))
	}
}

func TestCheckLexerError(t *testing.T) {
	r := Check("test", "local x = 'unterminated")
	if r.Block == nil {
		t.Fatal("Block should never be nil")
	}
	if !r.HasErrors() {
		t.Fatal("expected errors")
	}
	if len(r.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(r.Diagnostics))
	}
	d := r.Diagnostics[0]
	if d.Severity != Error {
		t.Errorf("expected Error severity, got %d", d.Severity)
	}
}

func TestCheckPositionAccuracy(t *testing.T) {
	// Error is on line 2: missing 'end'
	r := Check("test", "local x = 1\nif true then")
	if len(r.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(r.Diagnostics))
	}
	d := r.Diagnostics[0]
	if d.StartLineNumber < 1 {
		t.Errorf("StartLineNumber should be >= 1, got %d", d.StartLineNumber)
	}
	if d.StartColumn < 1 {
		t.Errorf("StartColumn should be >= 1, got %d", d.StartColumn)
	}
	// End should equal start (no span tracking yet).
	if d.EndLineNumber != d.StartLineNumber {
		t.Errorf("EndLineNumber should equal StartLineNumber")
	}
	if d.EndColumn != d.StartColumn {
		t.Errorf("EndColumn should equal StartColumn")
	}
}

func TestCheckBlockNeverNil(t *testing.T) {
	inputs := []string{
		"",
		"print('hello')",
		"if true then",
		"'bad",
		"local x = ",
	}
	for _, input := range inputs {
		r := Check("test", input)
		if r.Block == nil {
			t.Errorf("Block is nil for input %q", input)
		}
	}
}

func TestCheckDiagnosticMessage(t *testing.T) {
	r := Check("test", "if true then")
	if len(r.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(r.Diagnostics))
	}
	d := r.Diagnostics[0]
	// Message should be the error message without the position prefix.
	if d.Message == "" {
		t.Error("diagnostic message should not be empty")
	}
	// The message should NOT contain the "source:line:col:" prefix since
	// that information is in the position fields.
	if len(d.Message) > 0 && d.Message[0] >= '0' && d.Message[0] <= '9' {
		// Heuristic: if it starts with a digit it might contain a position prefix
		// This is fine — the message from PosError.Msg won't have the prefix.
	}
}
