// Package check provides Lua source diagnostics for editor integration.
//
// It parses partial/incomplete Lua source and returns diagnostics with
// positions matching Monaco's 1-based IMarkerData interface.
package check

import (
	"github.com/iceisfun/golua/v2/ast"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/token"
)

// Severity constants matching Monaco's MarkerSeverity.
const (
	Hint    = 1
	Info    = 2
	Warning = 4
	Error   = 8
)

// Diagnostic describes a single issue found in the source, with positions
// matching Monaco's 1-based IMarkerData interface.
type Diagnostic struct {
	Severity        int    `json:"severity"`
	Message         string `json:"message"`
	StartLineNumber int    `json:"startLineNumber"`
	StartColumn     int    `json:"startColumn"`
	EndLineNumber   int    `json:"endLineNumber"`
	EndColumn       int    `json:"endColumn"`
}

// Result holds the output of a Check call.
type Result struct {
	Block       *ast.Block   // partial AST, never nil
	Diagnostics []Diagnostic // list of diagnostics found
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
// The block is never nil — on error it contains all statements that parsed
// successfully before the first error.
func Check(source, input string) *Result {
	block, err := parser.ParsePartial(source, input)
	r := &Result{Block: block}
	if err == nil {
		return r
	}

	d := Diagnostic{
		Severity: Error,
		Message:  err.Error(),
	}

	if pe, ok := err.(*token.PosError); ok {
		d.Message = pe.Msg
		d.StartLineNumber = pe.Pos.Line
		d.StartColumn = pe.Pos.Column
		d.EndLineNumber = pe.Pos.Line
		d.EndColumn = pe.Pos.Column
	}

	r.Diagnostics = append(r.Diagnostics, d)
	return r
}
