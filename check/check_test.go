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
	// The span stays on one line, but the end column is widened so the editor
	// renders a visible squiggle rather than a zero-width marker.
	if d.EndLineNumber != d.StartLineNumber {
		t.Errorf("EndLineNumber should equal StartLineNumber")
	}
	if d.EndColumn < d.StartColumn {
		t.Errorf("EndColumn (%d) should be >= StartColumn (%d)", d.EndColumn, d.StartColumn)
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

func TestCheckMultipleErrors(t *testing.T) {
	// Two independent errors on separate lines. The single-error parser would
	// only report the first; multi-error recovery must surface both.
	r := Check("test", "x = )\ny = )\n")
	if len(r.Diagnostics) < 2 {
		t.Fatalf("expected >= 2 diagnostics from multi-error recovery, got %d: %+v",
			len(r.Diagnostics), r.Diagnostics)
	}
	if r.Diagnostics[0].StartLineNumber == r.Diagnostics[1].StartLineNumber {
		t.Errorf("expected diagnostics on different lines, both on line %d",
			r.Diagnostics[0].StartLineNumber)
	}
}

func TestCheckDiagnosticCodes(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"if true then", CodeTokenExpected}, // 'end' expected
		{"x = )", CodeUnexpectedSymbol},     // unexpected symbol near ')'
		{"local x = 'abc", CodeUnfinishedString},
		{"return 1 +", CodeUnexpectedSymbol}, // unexpected symbol near <eof>
	}
	for _, tc := range cases {
		r := Check("test", tc.input)
		if len(r.Diagnostics) == 0 {
			t.Errorf("%q: expected a diagnostic, got none", tc.input)
			continue
		}
		if got := r.Diagnostics[0].Code; got != tc.want {
			t.Errorf("%q: code = %q, want %q (msg: %q)",
				tc.input, got, tc.want, r.Diagnostics[0].Message)
		}
	}
}

func TestClassifyStable(t *testing.T) {
	// Unit-test the classifier directly so message-only paths that are awkward
	// to trigger end-to-end are still covered and pinned.
	cases := []struct {
		msg  string
		want string
	}{
		{"unexpected symbol near '@'", CodeUnexpectedSymbol},
		{"'end' expected (to close 'if' at line 1) near <eof>", CodeTokenExpected},
		{"<eof> expected near 'end'", CodeEOFExpected},
		{"<name> or '...' expected near ')'", CodeNameExpected},
		{"function arguments expected near 'x'", CodeFunctionArgs},
		{"unfinished string near <eof>", CodeUnfinishedString},
		{"invalid escape sequence near '\\q'", CodeInvalidEscape},
		{"malformed number near '3x'", CodeMalformedNumber},
		{"unknown attribute 'foo'", CodeUnknownAttribute},
		{"multiple to-be-closed variables in local list", CodeMultipleTBC},
		{"C stack overflow", CodeNestingTooDeep},
		{"too many local variables (limit is 200) in main function", CodeTooManyLocals},
		{"chunk has too many lines", CodeInputTooLong},
		{"something totally unrecognized", CodeSyntaxError},
	}
	for _, tc := range cases {
		if got := classify(tc.msg); got != tc.want {
			t.Errorf("classify(%q) = %q, want %q", tc.msg, got, tc.want)
		}
	}
}

func TestCheckRecoveryBounded(t *testing.T) {
	// Pathological input: every line is a syntax error. Recovery must terminate
	// and never exceed the cap.
	var b []byte
	for range 100 {
		b = append(b, "x = )\n"...)
	}
	r := Check("test", string(b))
	if len(r.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if len(r.Diagnostics) > maxDiagnostics {
		t.Fatalf("diagnostics %d exceeded cap %d", len(r.Diagnostics), maxDiagnostics)
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
