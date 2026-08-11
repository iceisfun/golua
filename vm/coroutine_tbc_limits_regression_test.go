// Package vm_test holds regression tests that need the standard library
// (which imports vm), so they cannot live in package vm itself.
package vm_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// newScriptVM compiles src and returns a fresh VM with captured output plus a
// runner. Compilation happens on the calling (test) goroutine so the runner is
// safe to launch on a worker goroutine.
func newScriptVM(t *testing.T, src string, opts ...vm.VMOption) (*vm.VM, func() error) {
	t.Helper()
	block, err := parser.Parse("test.lua", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test.lua", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New(append([]vm.VMOption{vm.WithCaptureOutput(true)}, opts...)...)
	stdlib.Open(v)
	return v, func() error {
		_, runErr := v.Run(proto)
		return runErr
	}
}

// runScript compiles and runs src on a fresh VM with captured output.
func runScript(t *testing.T, src string, opts ...vm.VMOption) (*vm.VM, error) {
	t.Helper()
	v, run := newScriptVM(t, src, opts...)
	return v, run()
}

// runScriptDeadline runs src on its own goroutine and fails if it has not
// returned within limit. A dead coroutine's pending <close> handler used to
// block the resumer forever on the coroutine's channels; in an embedded host
// (where Go's deadlock detector never fires because other goroutines are
// runnable) that is a permanently wedged worker.
//
// A regression leaves the worker parked on a channel, not spinning, so the
// t.Fatalf below leaks a blocked goroutine but no CPU.
func runScriptDeadline(t *testing.T, src string, limit time.Duration, opts ...vm.VMOption) (*vm.VM, error) {
	t.Helper()

	// Keep an unrelated goroutine runnable so the runtime cannot report the
	// hang as "all goroutines are asleep" — this mirrors a real server.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				time.Sleep(time.Millisecond)
			}
		}
	}()

	v, run := newScriptVM(t, src, opts...)
	done := make(chan error, 1)
	go func() { done <- run() }()
	select {
	case err := <-done:
		return v, err
	case <-time.After(limit):
		t.Fatalf("script did not return within %v (hung host goroutine)", limit)
		return nil, nil
	}
}

const yieldingCloseHandler = `setmetatable({}, {__close = function() coroutine.yield("c") end})`

// A coroutine that dies from an error keeps its <close> variables pending.
// coroutine.wrap closes them on the resumer's goroutine when it re-raises, so
// a yield from such a handler must raise the C-call-boundary error rather than
// block forever. Reference lua5.5.0 prints:
//
//	false	attempt to yield across a C-call boundary
//	still alive
func TestWrapPendingCloseYieldIsCatchable(t *testing.T) {
	src := `
local w = coroutine.wrap(function()
  local a <close> = ` + yieldingCloseHandler + `
  error("boom")
end)
local ok, err = pcall(w)
print(tostring(ok), tostring(err))
print("still alive")
`
	v, err := runScriptDeadline(t, src, 10*time.Second)
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	out := v.OutputLines()
	want := []string{"false\tattempt to yield across a C-call boundary", "still alive"}
	if len(out) != len(want) {
		t.Fatalf("output = %q, want %q", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("output[%d] = %q, want %q", i, out[i], want[i])
		}
	}
}

// Same pending-<close> state, reached through an explicit coroutine.close on
// the dead coroutine. Reference returns false plus the boundary error.
func TestExplicitClosePendingYieldIsCatchable(t *testing.T) {
	src := `
local co = coroutine.create(function()
  local a <close> = ` + yieldingCloseHandler + `
  error("boom")
end)
local ok = coroutine.resume(co)
print(tostring(ok))
local cok, cerr = coroutine.close(co)
print(tostring(cok), tostring(cerr))
print("still alive")
`
	v, err := runScriptDeadline(t, src, 10*time.Second)
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	out := v.OutputLines()
	want := []string{"false", "false\tattempt to yield across a C-call boundary", "still alive"}
	if len(out) != len(want) {
		t.Fatalf("output = %q, want %q", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("output[%d] = %q, want %q", i, out[i], want[i])
		}
	}
}

// An embedded host bounds the VM with a context deadline. The deadline is
// cooperative (checked between instructions), so it cannot rescue a goroutine
// parked on a channel — the call must return on its own.
func TestPendingCloseYieldRespectsContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	src := `
local w = coroutine.wrap(function()
  local a <close> = ` + yieldingCloseHandler + `
  error("boom")
end)
print(tostring((pcall(w))))
`
	start := time.Now()
	v, err := runScriptDeadline(t, src, 10*time.Second, vm.WithContext(ctx))
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("script took %v; it should finish without waiting on the deadline", elapsed)
	}
	if got := v.LastOutput(); got != "false" {
		t.Errorf("last output = %q, want %q", got, "false")
	}
}

// A non-yielding __close handler on a pending coroutine still runs normally.
func TestExplicitCloseStillRunsHandler(t *testing.T) {
	src := `
local co = coroutine.create(function()
  local a <close> = setmetatable({}, {__close = function() print("closed") end})
  coroutine.yield(1)
end)
coroutine.resume(co)
print(tostring(coroutine.close(co)))
`
	v, err := runScript(t, src)
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	out := v.OutputLines()
	want := []string{"closed", "true"}
	if len(out) != len(want) || out[0] != want[0] || out[1] != want[1] {
		t.Fatalf("output = %q, want %q", out, want)
	}
}

const instrLimit = 200000

// maxCoroutineSpawns bounds how many coroutines the shared-budget test lets the
// script start. Once the budget is genuinely shared, the first spinning
// coroutine consumes what is left of it, so only a handful ever start; without
// sharing the script spawns thousands (and total work grows as the square of
// the limit).
const maxCoroutineSpawns = 64

// Limits.MaxInstructions must bound the TOTAL work of a VM and every coroutine
// descended from it. When each coroutine got a fresh budget, these scripts never
// stopped spawning: every new coroutine started from zero again. coroutine.resume
// and coroutine.wrap drive the coroutine from separate code paths, so cover both.
func TestInstructionLimitSharedWithCoroutines(t *testing.T) {
	spinners := map[string]string{
		"resume": `while true do coroutine.resume(coroutine.create(function() tick() while true do end end)) end`,
		"wrap":   `while true do pcall(coroutine.wrap(function() tick() while true do end end)) end`,
	}
	for name, src := range spinners {
		t.Run(name, func(t *testing.T) {
			// The context is the test's escape hatch: cancelling it stops the VM
			// and every coroutine of the family at their next checkpoint, so a
			// regression fails fast instead of burning CPU (and leaves nothing
			// spinning behind).
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			v, run := newScriptVM(t, src,
				vm.WithContext(ctx),
				vm.WithLimits(vm.Limits{MaxInstructions: instrLimit}))
			var spawns atomic.Int64
			v.SetGlobal("tick", vm.NewNativeFunc(func(*vm.VM) int {
				if spawns.Add(1) > maxCoroutineSpawns {
					// Cancel rather than panic: a panic raised in a native called
					// from a coroutine body is caught by that coroutine's
					// protected call and only turns into a failed resume, so the
					// spawner loop would keep going.
					cancel()
				}
				return 0
			}))

			done := make(chan error, 1)
			go func() { done <- run() }()
			var err error
			select {
			case err = <-done:
			case <-time.After(60 * time.Second):
				t.Fatalf("script did not stop; spawned %d coroutines", spawns.Load())
			}
			if err == nil {
				t.Fatal("expected the instruction limit to be exceeded")
			}
			if !strings.Contains(err.Error(), "instruction limit exceeded") {
				t.Fatalf("error = %v (spawned %d coroutines), want instruction limit exceeded",
					err, spawns.Load())
			}
			if got := spawns.Load(); got > maxCoroutineSpawns {
				t.Errorf("spawned %d coroutines, want <= %d", got, maxCoroutineSpawns)
			}
			// Each VM in the family may overshoot by one checkpoint before its
			// own check fires, so allow a small slack over the budget.
			if got := v.InstructionCount(); got > instrLimit+maxCoroutineSpawns {
				t.Errorf("instruction count = %d, want <= %d (budget not shared)",
					got, instrLimit+maxCoroutineSpawns)
			}
		})
	}
}

// Long-lived coroutines that yield must charge their work to the family budget
// at every handoff, not only when they die. A pool of coroutines resumed
// round-robin spreads the work so thinly that no single coroutine exhausts a
// per-VM budget of its own: unshared, the pool runs two orders of magnitude
// longer than the configured limit before the resumer alone trips it.
func TestInstructionLimitSharedAcrossYields(t *testing.T) {
	// maxRounds is a generous bound on the coroutine-side work units a shared
	// budget permits (~200000/300 = 660 of them).
	const maxRounds = 5000

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	src := `
local pool = {}
for i = 1, 200 do
  pool[i] = coroutine.create(function()
    while true do
      for j = 1, 100 do end
      round()
      coroutine.yield()
    end
  end)
end
while true do
  for i = 1, 200 do coroutine.resume(pool[i]) end
end
`
	v, run := newScriptVM(t, src,
		vm.WithContext(ctx),
		vm.WithLimits(vm.Limits{MaxInstructions: instrLimit}))
	var rounds atomic.Int64
	v.SetGlobal("round", vm.NewNativeFunc(func(*vm.VM) int {
		if rounds.Add(1) > 20*maxRounds {
			cancel() // escape hatch, see TestInstructionLimitSharedWithCoroutines
		}
		return 0
	}))
	err := run()
	if err == nil || !strings.Contains(err.Error(), "instruction limit exceeded") {
		t.Fatalf("error = %v (%d rounds), want instruction limit exceeded", err, rounds.Load())
	}
	if got := rounds.Load(); got > maxRounds {
		t.Errorf("coroutines ran %d work units, want <= %d (budget not shared across yields)",
			got, maxRounds)
	}
	// Every VM in the family may overshoot by one checkpoint before its own
	// check fires, so allow slack for the whole pool.
	if got := v.InstructionCount(); got > instrLimit+201 {
		t.Errorf("instruction count = %d, want <= %d (budget not shared)", got, instrLimit+201)
	}
}

// A dead coroutine's pending <close> handlers run on the closer's goroutine
// against the coroutine's VM, so that work has to reach the shared budget too —
// otherwise erroring and closing coroutine after coroutine buys an unbounded
// amount of __close time for a handful of resumer instructions each.
func TestInstructionLimitCoversPendingCloseHandlers(t *testing.T) {
	// maxHandlers bounds the __close handlers a shared budget permits
	// (~200000/10000 = 20 of them).
	const maxHandlers = 500

	// The two ways to reach a dead coroutine's pending handlers.
	const heavyCloseVar = `
    local a <close> = setmetatable({}, {__close = function()
      handler()
      for i = 1, 10000 do end
    end})`
	scripts := map[string]string{
		"close": `
while true do
  local co = coroutine.create(function()` + heavyCloseVar + `
    error("boom")
  end)
  coroutine.resume(co)
  coroutine.close(co)
end
`,
		"wrap": `
while true do
  local w = coroutine.wrap(function()` + heavyCloseVar + `
    error("boom")
  end)
  pcall(w)
end
`,
	}

	for name, src := range scripts {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			v, run := newScriptVM(t, src,
				vm.WithContext(ctx),
				vm.WithLimits(vm.Limits{MaxInstructions: instrLimit}))
			var handlers atomic.Int64
			v.SetGlobal("handler", vm.NewNativeFunc(func(*vm.VM) int {
				if handlers.Add(1) > 20*maxHandlers {
					cancel() // escape hatch, see TestInstructionLimitSharedWithCoroutines
				}
				return 0
			}))
			err := run()
			if err == nil || !strings.Contains(err.Error(), "instruction limit exceeded") {
				t.Fatalf("error = %v (%d handlers), want instruction limit exceeded",
					err, handlers.Load())
			}
			if got := handlers.Load(); got > maxHandlers {
				t.Errorf("ran %d __close handlers, want <= %d (their work escapes the budget)",
					got, maxHandlers)
			}
		})
	}
}

// A script that never creates a coroutine must still trip at exactly the same
// point as before the budget became shared.
func TestInstructionLimitSingleVMUnchanged(t *testing.T) {
	v, err := runScript(t, `local i = 0 while true do i = i + 1 end`,
		vm.WithLimits(vm.Limits{MaxInstructions: instrLimit}))
	if err == nil {
		t.Fatal("expected the instruction limit to be exceeded")
	}
	if !strings.Contains(err.Error(), "instruction limit exceeded") {
		t.Fatalf("error = %v, want instruction limit exceeded", err)
	}
	if got := v.InstructionCount(); got != instrLimit+1 {
		t.Errorf("instruction count = %d, want %d", got, instrLimit+1)
	}
	v.ResetInstructionCount()
	if got := v.InstructionCount(); got != 0 {
		t.Errorf("instruction count after reset = %d, want 0", got)
	}
}

// The shared budget is charged from the coroutine's goroutine and read from the
// resumer's, so make sure the resume/yield handoff keeps it consistent.
func TestInstructionCountVisibleAfterCoroutineRuns(t *testing.T) {
	src := `
local co = coroutine.create(function()
  for i = 1, 1000 do end
  coroutine.yield()
end)
coroutine.resume(co)
coroutine.close(co)
`
	v, err := runScript(t, src, vm.WithLimits(vm.Limits{MaxInstructions: 1 << 30}))
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	if v.InstructionCount() < 1000 {
		t.Errorf("instruction count = %d, want the coroutine's work included",
			v.InstructionCount())
	}
}

// abandonedCoroutineScript resumes a coroutine that never yields and whose body
// sits inside a native call (no VM checkpoints) when the context deadline
// fires, so the resumer always takes the give-up branch and returns while the
// coroutine goroutine is still running. From that point the two goroutines
// never synchronise again: anything they share must be safe without a handoff.
const abandonedCoroutineScript = `
local co = coroutine.create(function()
  local x = 0
  while true do x = x + 1 spin() end
end)
coroutine.resume(co)
local i = 0
while true do i = i + 1 end
`

// abandonRunner builds a VM running abandonedCoroutineScript under a short
// deadline, plus the spin() native that keeps the coroutine off its checkpoints
// while the deadline fires.
func abandonRunner(t *testing.T, ctx context.Context) (*vm.VM, func() error) {
	t.Helper()
	v, run := newScriptVM(t, abandonedCoroutineScript,
		vm.WithContext(ctx),
		vm.WithLimits(vm.Limits{MaxInstructions: 1 << 40}))
	v.SetGlobal("spin", vm.NewNativeFunc(func(*vm.VM) int {
		deadline := time.Now().Add(4 * time.Millisecond)
		for time.Now().Before(deadline) {
		}
		return 0
	}))
	return v, run
}

// The instruction budget is shared by the whole coroutine family, and a
// context-cancelled resume abandons a running coroutine goroutine, so the
// sharing must be race-free. Run under -race.
func TestAbandonedCoroutineBudgetRaceHostRead(t *testing.T) {
	for i := 0; i < 50; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		v, run := abandonRunner(t, ctx)
		err := run()
		if err == nil || !strings.Contains(err.Error(), "execution interrupted") {
			cancel()
			t.Fatalf("run error = %v, want the context deadline to interrupt it", err)
		}
		// The abandoned coroutine goroutine has charged the shared budget
		// without ever handing it back through a channel.
		_ = v.InstructionCount()
		cancel()
	}
}

// The documented sandbox recovery pattern (SetContext + ResetInstructionCount +
// reuse) keeps the host executing on the same VM while an abandoned coroutine
// goroutine is still charging the shared budget. Run under -race.
func TestAbandonedCoroutineBudgetRaceVMReuse(t *testing.T) {
	block, err := parser.Parse("reuse.lua", `local i = 0 for k = 1, 20000 do i = i + k end`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	second, err := compiler.Compile("reuse.lua", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	for i := 0; i < 50; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		v, run := abandonRunner(t, ctx)
		err := run()
		if err == nil || !strings.Contains(err.Error(), "execution interrupted") {
			cancel()
			t.Fatalf("run error = %v, want the context deadline to interrupt it", err)
		}
		// Reuse the VM exactly as an embedder recovering from a timeout does.
		v.SetContext(context.Background())
		v.ResetInstructionCount()
		if _, err := v.Run(second); err != nil {
			cancel()
			t.Fatalf("reuse run failed: %v", err)
		}
		cancel()
	}
}

// compileScript compiles src on the calling (test) goroutine so the resulting
// proto can be run repeatedly on fresh VMs.
func compileScript(t *testing.T, name, src string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse(name, src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

// A resume that gives up on a cancelled context abandons the coroutine
// goroutine; the coroutine then dies on its own, publishing status=dead under
// co.mu BEFORE it finishes with its VM (final instruction-budget sync, stack
// release) and before it closes doneCh. The host meanwhile reuses the VM (the
// documented timeout-recovery pattern) and the script closes the now-dead
// coroutine, which takes over the coroutine's VM from the closer's goroutine.
//
// "Dead" alone is therefore not a valid handoff of that VM: coroutine.close
// must wait for doneCh first. Without the wait the closer tears the coroutine
// VM's plain (instrCount, instrSynced) pair, which either double-charges the
// family budget (a spurious "instruction limit exceeded" for a legitimate
// script) or rewinds it. Run under -race.
func TestCloseDeadAbandonedCoroutineBudgetRace(t *testing.T) {
	resume := compileScript(t, "resume.lua", `
co = coroutine.create(function()
  spin()
  error("boom")
end)
return coroutine.resume(co)
`)
	closeCo := compileScript(t, "close.lua", `return coroutine.close(co)`)

	abandoned := 0
	for i := 0; i < 400; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		v := vm.New(vm.WithContext(ctx), vm.WithLimits(vm.Limits{MaxInstructions: 1 << 40}))
		stdlib.Open(v)
		// Sweep the spin duration across the deadline so the coroutine dies
		// just before, during and just after the resumer gives up.
		spinFor := time.Duration(2050+(i%200)*3) * time.Microsecond
		v.SetGlobal("spin", vm.NewNativeFunc(func(*vm.VM) int {
			deadline := time.Now().Add(spinFor)
			for time.Now().Before(deadline) {
			}
			return 0
		}))
		res, _ := v.Run(resume)
		if len(res) >= 1 && !res[0].AsBool() {
			abandoned++
		}
		// Recover the VM for reuse exactly as an embedder does, then close the
		// dead coroutine from this goroutine.
		v.SetContext(context.Background())
		v.ResetInstructionCount()
		// The close may legitimately fail ("cannot close a running coroutine")
		// when the abandoned goroutine is still spinning; the interesting
		// iterations are the ones where it has just gone dead.
		_, _ = v.Run(closeCo)
		_ = v.InstructionCount()
		cancel()
	}
	if abandoned == 0 {
		t.Skip("no resume was interrupted by the deadline; machine too slow to exercise the race")
	}
	t.Logf("abandoned %d/400 resumes", abandoned)
}
