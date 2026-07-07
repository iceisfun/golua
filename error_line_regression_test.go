package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// runLuaExpectError compiles and runs source, returning the runtime error string
// (empty if none). Used to assert error line attribution.
func runLuaExpectError(t *testing.T, source string) string {
	t.Helper()
	block, err := parser.Parse("errline", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("errline", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New(vm.WithCaptureOutput(true))
	stdlib.Open(v)
	_, err = v.Run(proto)
	if err == nil {
		return ""
	}
	return err.Error()
}

// an indexed/field-assignment store error must report the RHS
// (explist) end line, not the target's line — matching reference restassign.
func TestStoreErrorLine(t *testing.T) {
	// single-assign, RHS on line 3
	got := runLuaExpectError(t, "local t\nt.x =\n  5\n")
	if !strings.Contains(got, "]:3:") {
		t.Fatalf("single store line: got %q want line 3", got)
	}
	// multi-assign, explist ends on line 6
	got = runLuaExpectError(t, "local t={}\nlocal u\nt.a,\nu.b =\n  1,\n  2\n")
	if !strings.Contains(got, "]:6:") {
		t.Fatalf("multi store line: got %q want line 6", got)
	}
}

// an error raised inside a generic-for iterator must report the
// iterator explist's start line (reference captures the line after 'in').
func TestForInIteratorErrorLine(t *testing.T) {
	got := runLuaExpectError(t, "for k in\n  (function() error('boom') end),\n  nil,\n  nil\ndo end\n")
	// The error site reports the iterator explist start line (2), not the last
	// iterator (4). The "in for iterator" traceback frame is likewise attributed
	// to line 2 (verified at the CLI level).
	if !strings.Contains(got, "]:2: boom") {
		t.Fatalf("for-in iterator line: got %q want line 2", got)
	}
}
