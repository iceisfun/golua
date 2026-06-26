package golua_test

// Goroutine-lifecycle characterization for coroutines. golua backs each
// coroutine with a goroutine; a SUSPENDED coroutine is a goroutine parked on its
// resume channel. Driving a coroutine to completion or coroutine.close()-ing it
// reaps that goroutine. An *abandoned suspended* coroutine cannot be reaped — Go
// cannot kill a blocked goroutine and the parked goroutine is itself a GC root —
// so it leaks until the process exits. Reference Lua collects abandoned
// suspended coroutines; this divergence is documented in
// wontfix/coroutine-goroutine-leak. Mitigation: drive coroutines to completion
// or coroutine.close() them.
//
// This test pins the SUPPORTED paths (complete / close reap the goroutine) as
// regression guards, and records the abandoned-leak as the known limitation.

import (
	"runtime"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func goroutinesAfterSettle() int {
	runtime.GC()
	time.Sleep(150 * time.Millisecond)
	runtime.GC()
	return runtime.NumGoroutine()
}

func runCoLifecycle(t *testing.T, src string) {
	t.Helper()
	block, err := parser.Parse("=colife", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p, err := compiler.Compile("=colife", block)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	v := vm.New()
	stdlib.Open(v)
	if _, err := v.Run(p); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestCoroutineGoroutineLifecycle(t *testing.T) {
	const N = 500
	body := "function() for j = 1, 50 do coroutine.yield(j) end end"

	base := goroutinesAfterSettle()

	// Driven to completion -> goroutine reaped.
	runCoLifecycle(t, "for i=1,"+itoa(N)+" do "+
		"local co = coroutine.create("+body+"); "+
		"coroutine.resume(co); "+
		"while coroutine.status(co) ~= 'dead' do coroutine.resume(co) end end")
	if g := goroutinesAfterSettle(); g > base+50 {
		t.Fatalf("drive-to-dead leaked goroutines: %d -> %d (regression: completed coroutines must reap)", base, g)
	}

	// Explicitly closed -> goroutine reaped.
	runCoLifecycle(t, "for i=1,"+itoa(N)+" do "+
		"local co = coroutine.create("+body+"); "+
		"coroutine.resume(co); coroutine.close(co) end")
	if g := goroutinesAfterSettle(); g > base+50 {
		t.Fatalf("coroutine.close leaked goroutines: %d -> %d (regression: closed coroutines must reap)", base, g)
	}

	// Abandoned suspended -> KNOWN to leak one goroutine each (documented).
	runCoLifecycle(t, "for i=1,"+itoa(N)+" do "+
		"local co = coroutine.create("+body+"); coroutine.resume(co) end")
	leaked := goroutinesAfterSettle() - base
	t.Logf("abandoned-suspended coroutines: %d created, ~%d goroutines retained "+
		"(known limitation — see wontfix/coroutine-goroutine-leak; mitigation: "+
		"coroutine.close or run to completion)", N, leaked)
	if leaked < N/2 {
		// If a future change makes abandoned coroutines reapable, this turns the
		// documentation into a real win — update the test to assert no leak.
		t.Logf("NOTE: abandoned coroutines no longer leak (%d retained) — the "+
			"goroutine-leak limitation may have been fixed; tighten this test.", leaked)
	}
}

// itoa avoids importing strconv just for the loop bounds in test sources.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
