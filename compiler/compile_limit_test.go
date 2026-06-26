package compiler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/parser"
)

// TestTooManyLocalsStopsAtLimit is a regression guard for the compile-time
// O(n^2) DoS where, after recording a "too many local variables" error, the
// compiler kept walking every remaining statement (each doing O(active-locals)
// work) instead of stopping like reference Lua's longjmp-on-error. A chunk with
// 200k local declarations took ~18s (O(n^2)) before the fix and ~0.1s after
// (it stops at the 201st local). The error itself is unchanged.
func TestTooManyLocalsStopsAtLimit(t *testing.T) {
	const n = 200000
	var sb strings.Builder
	sb.Grow(n * 16)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "local _x%d = %d\n", i, i)
	}
	sb.WriteString("return 1\n")

	block, err := parser.Parse("<test>", sb.String())
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	start := time.Now()
	_, cerr := Compile("<test>", block)
	elapsed := time.Since(start)

	if cerr == nil {
		t.Fatal("expected a 'too many local variables' compile error")
	}
	if !strings.Contains(cerr.Error(), "too many local variables (limit is 200)") {
		t.Fatalf("unexpected error: %v", cerr)
	}
	// Generous bound: post-fix ~0.1s, pre-fix (O(n^2)) many seconds.
	if elapsed > 5*time.Second {
		t.Fatalf("compiling %d locals took %v — O(n^2) regression (compiler must stop at first error)", n, elapsed)
	}
}
