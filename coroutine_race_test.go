package golua_test

import (
	"sync"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// compileLua compiles Lua source into a Proto for execution.
func compileLua(t *testing.T, source, name string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

// TestCoroutineResumeCloseRace verifies that concurrent resume and close on
// the same coroutine does not panic with "send on closed channel".
//
// The race: coResume sends on co.resumeCh (line 216) while coClose closes
// co.resumeCh (line 738). If these operations interleave, Go panics with
// "send on closed channel". This test shares a coroutine thread table
// between two VMs so that resume and close execute on separate goroutines.
//
// Run with: go test -race -run TestCoroutineResumeCloseRace -count=1
func TestCoroutineResumeCloseRace(t *testing.T) {
	// Pre-compile the Lua fragments once.
	setupProto := compileLua(t, `
		co = coroutine.create(function()
			while true do coroutine.yield() end
		end)
		coroutine.resume(co) -- first resume; coroutine yields and is now suspended
	`, "setup")

	resumeProto := compileLua(t, `return coroutine.resume(co)`, "resume")
	closeProto := compileLua(t, `return coroutine.close(co)`, "close")

	// Run enough iterations to trigger the race with high probability.
	// With -race, even a single interleaving is detected.
	const iterations = 200

	for i := 0; i < iterations; i++ {
		// VM1 owns the coroutine.
		v1 := vm.New()
		stdlib.Open(v1)
		if _, err := v1.Run(setupProto); err != nil {
			t.Fatalf("setup error: %v", err)
		}

		// Extract the thread table and share it with VM2.
		coThread := v1.GetGlobal("co")

		v2 := vm.New()
		stdlib.Open(v2)
		v2.SetGlobal("co", coThread)

		// Race: v1 resumes while v2 closes the same coroutine.
		var (
			wg         sync.WaitGroup
			resumeErr  error
			closeErr   error
			resumePanic any
			closePanic  any
		)
		wg.Add(2)

		go func() {
			defer wg.Done()
			defer func() { resumePanic = recover() }()
			_, resumeErr = v1.Run(resumeProto)
		}()

		go func() {
			defer wg.Done()
			defer func() { closePanic = recover() }()
			_, closeErr = v2.Run(closeProto)
		}()

		wg.Wait()

		// Neither goroutine should have panicked.
		if resumePanic != nil {
			t.Fatalf("iteration %d: resume panicked: %v", i, resumePanic)
		}
		if closePanic != nil {
			t.Fatalf("iteration %d: close panicked: %v", i, closePanic)
		}

		// Errors from Lua are acceptable (e.g. "cannot resume dead coroutine",
		// "cannot close a running coroutine") — those are graceful failures.
		// The test only asserts no unrecovered panic escaped.
		_ = resumeErr
		_ = closeErr
	}
}
