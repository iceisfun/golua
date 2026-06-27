// Package check provides Lua source diagnostics for editor integration.
//
// It parses partial/incomplete Lua source and returns diagnostics with
// positions matching Monaco's 1-based IMarkerData interface. Each diagnostic
// also carries a stable, machine-readable [Diagnostic.Code] so an editor can
// group, filter, or link diagnostics independently of the human-readable
// message text.
//
// Unlike the reference parser — which stops at the first syntax error — Check
// performs *multi-error recovery*: after reporting an error it blanks the
// offending lines and re-parses the remainder, surfacing several independent
// problems in one pass. The conformance-critical parser is never modified; all
// recovery happens at this layer (see [Check]).
package check

import (
	"strings"
	"unicode/utf8"

	"github.com/iceisfun/golua/ast"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/token"
)

// Severity constants matching Monaco's MarkerSeverity.
const (
	Hint    = 1
	Info    = 2
	Warning = 4
	Error   = 8
)

// Diagnostic codes are stable, machine-readable identifiers for a class of
// problem. They are derived from the parser/lexer message text but are
// guaranteed stable across message-wording changes, so editor tooling can key
// off them (code-action links, quick-fix routing, severity overrides, etc.).
const (
	CodeUnexpectedSymbol = "unexpected-symbol"
	CodeTokenExpected    = "token-expected"
	CodeNameExpected     = "name-expected"
	CodeEOFExpected      = "eof-expected"
	CodeFunctionArgs     = "function-arguments-expected"
	CodeUnfinishedString = "unfinished-string"
	CodeInvalidEscape    = "invalid-escape"
	CodeMalformedNumber  = "malformed-number"
	CodeUnknownAttribute = "unknown-attribute"
	CodeMultipleTBC      = "multiple-to-be-closed"
	CodeNestingTooDeep   = "nesting-too-deep"
	CodeTooManyLocals    = "too-many-local-variables"
	CodeInputTooLong     = "input-too-long"
	CodeSyntaxError      = "syntax-error"
)

// maxDiagnostics caps how many problems a single Check reports. Multi-error
// recovery (see [Check]) can otherwise cascade on pathological input; ten
// independent diagnostics is plenty for an editor gutter and keeps the work
// bounded.
const maxDiagnostics = 10

// Diagnostic describes a single issue found in the source, with positions
// matching Monaco's 1-based IMarkerData interface.
type Diagnostic struct {
	Severity        int    `json:"severity"`
	Code            string `json:"code,omitempty"`
	Message         string `json:"message"`
	StartLineNumber int    `json:"startLineNumber"`
	StartColumn     int    `json:"startColumn"`
	EndLineNumber   int    `json:"endLineNumber"`
	EndColumn       int    `json:"endColumn"`
}

// Result holds the output of a Check call.
type Result struct {
	Block       *ast.Block   // partial AST, never nil
	Diagnostics []Diagnostic // list of diagnostics found (capped at maxDiagnostics)
}

// HasErrors reports whether any diagnostic has Error severity.
func (r *Result) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// Check parses the given Lua source and returns a partial AST and diagnostics.
//
// The block is never nil — it is the partial AST from the *first* parse,
// containing every statement that parsed before the first error. Editor
// features that consume the AST (symbols, hover) should use it as before.
//
// Check then performs multi-error recovery to populate Diagnostics: after each
// error it blanks every source line up to and including the error line
// (preserving newlines so reported positions stay absolute) and re-parses the
// remainder, collecting the next independent diagnostic. This is a resync at
// the check layer — the parser itself is unmodified and still single-error.
//
// The trade-off of line-level recovery: blanking the opening line of a
// multi-line construct can produce a cascade diagnostic on a later line. The
// count is capped at maxDiagnostics and recovery stops as soon as a re-parse
// succeeds, makes no forward progress, or hits an unpositioned error.
func Check(source, input string) *Result {
	// First pass: the real parse. Its block is what AST consumers rely on.
	block, err := parser.ParsePartial(source, input)
	r := &Result{Block: block}
	if err == nil {
		return r
	}

	d := toDiagnostic(err)
	r.Diagnostics = append(r.Diagnostics, d)

	lines := strings.Split(input, "\n")
	blankedThrough := 0 // count of leading lines already blanked
	for len(r.Diagnostics) < maxDiagnostics {
		errLine := d.StartLineNumber
		if errLine < 1 || errLine > len(lines) {
			break // unpositioned error (e.g. bare "C stack overflow") — stop
		}
		if errLine <= blankedThrough {
			break // re-parse reported a line we already blanked — no progress
		}
		for i := blankedThrough; i < errLine; i++ {
			lines[i] = "" // blank through the error line (0-indexed)
		}
		blankedThrough = errLine

		if _, err = parser.ParsePartial(source, strings.Join(lines, "\n")); err == nil {
			break
		}
		d = toDiagnostic(err)
		r.Diagnostics = append(r.Diagnostics, d)
	}
	return r
}

// toDiagnostic converts a parser/lexer error into a Diagnostic, assigning a
// stable code and a visible span.
func toDiagnostic(err error) Diagnostic {
	d := Diagnostic{Severity: Error, Message: err.Error()}

	pe, ok := err.(*token.PosError)
	if !ok {
		d.Code = CodeSyntaxError
		return d
	}

	d.Message = pe.Msg
	d.Code = classify(pe.Msg)
	d.StartLineNumber = pe.Pos.Line
	d.StartColumn = pe.Pos.Column
	d.EndLineNumber = pe.Pos.Line
	d.EndColumn = pe.Pos.Column

	// Widen the span so the editor renders a visible squiggle rather than a
	// zero-width marker. Prefer the width of the "near '<token>'" clause the
	// parser appends (the offending token sits at pe.Pos); otherwise fall back
	// to a single column.
	if pe.Pos.Column > 0 {
		if w := nearTokenWidth(pe.Msg); w > 0 {
			d.EndColumn = pe.Pos.Column + w
		} else {
			d.EndColumn = pe.Pos.Column + 1
		}
	}
	return d
}

// classify maps a parser/lexer message to a stable diagnostic code. Matching is
// substring-based so it survives minor wording differences (and the small
// 5.4/5.5 message divergences). Order matters: specific cases precede the
// generic "expected" / "syntax error" fallbacks.
func classify(msg string) string {
	switch {
	case strings.Contains(msg, "C stack overflow"):
		return CodeNestingTooDeep
	case strings.Contains(msg, "too many local variables"):
		return CodeTooManyLocals
	case strings.Contains(msg, "unexpected symbol"):
		return CodeUnexpectedSymbol
	case strings.Contains(msg, "<eof> expected"):
		return CodeEOFExpected
	case strings.Contains(msg, "function arguments expected"):
		return CodeFunctionArgs
	case strings.Contains(msg, "<name>"), strings.Contains(msg, "'=' or 'in' expected"):
		return CodeNameExpected
	case strings.Contains(msg, "unfinished string"), strings.Contains(msg, "unfinished long"):
		return CodeUnfinishedString
	case strings.Contains(msg, "escape"),
		strings.Contains(msg, "hexadecimal digit expected"),
		strings.Contains(msg, "UTF-8 value too large"),
		strings.Contains(msg, "missing '{'"),
		strings.Contains(msg, "missing '}'"):
		return CodeInvalidEscape
	case strings.Contains(msg, "malformed number"):
		return CodeMalformedNumber
	case strings.Contains(msg, "unknown attribute"):
		return CodeUnknownAttribute
	case strings.Contains(msg, "multiple to-be-closed"):
		return CodeMultipleTBC
	case strings.Contains(msg, "too many lines"), strings.Contains(msg, "too long"):
		return CodeInputTooLong
	case strings.Contains(msg, "expected"):
		return CodeTokenExpected
	default:
		return CodeSyntaxError
	}
}

// nearTokenWidth returns the rune width of the token in a trailing
// "near '<token>'" clause, or 0 if there is no usable single-line token (e.g.
// "near <eof>", which is unquoted). Used only to size the diagnostic span.
func nearTokenWidth(msg string) int {
	const marker = " near '"
	i := strings.LastIndex(msg, marker)
	if i < 0 {
		return 0
	}
	rest := msg[i+len(marker):]
	j := strings.LastIndex(rest, "'")
	if j <= 0 {
		return 0
	}
	tok := rest[:j]
	if tok == "" || strings.ContainsAny(tok, "\n\r") {
		return 0
	}
	if n := utf8.RuneCountInString(tok); n > 0 && n <= 80 {
		return n
	}
	return 0
}
