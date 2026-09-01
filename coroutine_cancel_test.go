package golua_test

// A coroutine runs on its own goroutine, but against the state its VM family
// shares: globals, tables, the string cache. Whoever abandons a *running*
// coroutine therefore has to wait for it to stop first. Two paths abandon one:
// a resume that gives up because the VM's context was cancelled, and VM.Close
// reaping the coroutines that were left suspended. These tests pin the
// invariant on both, plus the requirement that VM.Close returns whatever
// context it is handed.

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// TestCancelledResumeWaitsForCoroutine checks that when the VM's context fires
// while a coroutine is running, the resume does not hand control back to the
// host until the coroutine's goroutine has actually stopped.
//
// The coroutine spends its time inside a native call, which is the one place
// the VM cannot poll the context, so "has it stopped?" is directly observable:
// the counter the native bumps must not move once VM.Run has returned.
func TestCancelledResumeWaitsForCoroutine(t *testing.T) {
	var ticks atomic.Int64

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	v := vm.New(vm.WithContext(ctx))
	stdlib.Open(v)
	v.SetGlobal("work", vm.NewNativeFunc(func(*vm.VM) int {
		time.Sleep(250 * time.Millisecond)
		ticks.Add(1)
		return 0
	}))

	proto := compileLua(t, `
		local co = coroutine.create(function()
			while true do work() end
		end)
		coroutine.resume(co)
	`, "=cancelresume")
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("run: %v", err)
	}

	atReturn := ticks.Load()
	time.Sleep(700 * time.Millisecond)
	if final := ticks.Load(); final != atReturn {
		t.Fatalf("coroutine goroutine was still running when its cancelled resume "+
			"returned to the host: %d work() calls at return, %d after settling",
			atReturn, final)
	}
	if atReturn == 0 {
		t.Fatalf("test did not exercise the cancellation path: work() never ran")
	}
}

// TestCancelledResumeSharedTable is the end-to-end shape of the same defect: a
// coroutine and its resumer writing one Lua table while the VM's context
// expires. If the resumer walks away from a running coroutine, both goroutines
// write the table's backing map at once.
//
// This one is a stress probe: it reads the sharpest signal off the race
// detector, and without it the collision surfaces only if it lands hard enough
// to trip Go's uncatchable "concurrent map writes". The assertion it does make
// on its own is that the cancellation path was reached at all;
// TestCancelledResumeWaitsForCoroutine and
// TestCancelledResumeDoesNotOverlapCoroutine are the checks that hold without
// -race.
func TestCancelledResumeSharedTable(t *testing.T) {
	// Straight-line bursts: the VM polls its context at loop back-edges and at
	// calls, so a run of plain stores is where an abandoned coroutine keeps
	// going the longest.
	var coBody, mainBody strings.Builder
	for i := 0; i < 400; i++ {
		coBody.WriteString("shared['c" + strconv.Itoa(i) + "'] = n\n")
	}
	for i := 0; i < 400; i++ {
		mainBody.WriteString("shared['m" + strconv.Itoa(i) + "'] = i\n")
	}
	src := `
		shared = {}
		local co = coroutine.create(function()
			local n = 0
			while true do
				n = n + 1
				` + coBody.String() + `
			end
		end)
		coroutine.resume(co)
		for i = 1, 100000 do
			` + mainBody.String() + `
		end
	`

	proto := compileLua(t, src, "=sharedtable")
	interrupted := 0
	for attempt := 0; attempt < 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
		v := vm.New(vm.WithContext(ctx))
		stdlib.Open(v)
		_, err := v.Run(proto)
		cancel()
		if err != nil && strings.Contains(err.Error(), "execution interrupted") {
			interrupted++
		}
	}
	if interrupted == 0 {
		t.Fatalf("no attempt reached the cancellation path: raise the loop count " +
			"or lower the deadline, this test is checking nothing as it stands")
	}
}

// TestCancelledResumeDoesNotOverlapCoroutine is the same invariant as
// TestCancelledResumeSharedTable with the timing taken out of it, so it holds
// without the race detector.
//
// The coroutine cancels the VM's context itself, which pins the moment the
// resumer gives up: part-way through a run of straight-line stores, where the
// coroutine has not reached an interrupt poll and so is still writing the shared
// table. The main chunk then reads one of that run's counters twice, far apart
// but with no interrupt poll of its own in between. Both reads happen after the
// resume returned, so if the resume waited for the coroutine to stop they are
// necessarily equal, and if it did not they straddle the abandoned coroutine's
// stores.
func TestCancelledResumeDoesNotOverlapCoroutine(t *testing.T) {
	var coBody, probe strings.Builder
	for i := 0; i < 20000; i++ {
		coBody.WriteString("shared[1] = shared[1] + 1\n")
	}
	for i := 0; i < 10000; i++ {
		probe.WriteString("shared[2] = shared[2] + 1\n")
	}
	proto := compileLua(t, `
		shared = {0, 0}
		local co = coroutine.create(function()
			cancelnow()
			while true do
`+coBody.String()+`
			end
		end)
		coroutine.resume(co)
		local a = shared[1]
`+probe.String()+`
		local b = shared[1]
		skew = b - a
	`, "=resumeoverlap")

	// Where the coroutine's run of stores lands relative to the resumer waking
	// up is down to the scheduler, so one attempt can miss the window; a resume
	// that waits properly has skew 0 on every attempt, whatever the scheduler
	// does.
	for attempt := 0; attempt < 25; attempt++ {
		ctx, cancel := context.WithCancel(context.Background())
		v := vm.New(vm.WithContext(ctx))
		stdlib.Open(v)
		v.SetGlobal("cancelnow", vm.NewNativeFunc(func(*vm.VM) int {
			cancel()
			return 0
		}))
		_, _ = v.Run(proto)
		skew := v.GetGlobal("skew")
		cancel()

		if skew.IsNil() {
			t.Fatalf("attempt %d: the main chunk never reached the probe, so "+
				"nothing was measured", attempt)
		}
		if got := skew.String(); got != "0" {
			t.Fatalf("attempt %d: the coroutine's goroutine went on writing the "+
				"family's shared table after its cancelled resume returned: the "+
				"counter moved by %s between two reads made from the resumer",
				attempt, got)
		}
	}
}

// Termination covers what a withdrawn VM goes on to start. VM.Close terminates
// a coroutine whose <close> handler will not stop, and that handler is running
// Lua, so it can still reach coroutine.create and set another goroutine going
// against the same shared state. A coroutine created by a terminated VM has to
// start out terminated too, or it is bounded by nothing but the hard grace.
func TestCoroutineOfTerminatedVMStartsTerminated(t *testing.T) {
	parent := vm.New()
	live := vm.NewCoroutineVM(parent, make(chan []vm.Value), make(chan []vm.Value), 1)
	if live.Terminated() {
		t.Fatalf("a coroutine of a live VM started out terminated")
	}

	parent.Terminate()
	co := vm.NewCoroutineVM(parent, make(chan []vm.Value), make(chan []vm.Value), 2)
	if !co.Terminated() {
		t.Fatalf("a coroutine created by a terminated VM started out running")
	}

	// And the flag has to be the one execution actually looks at.
	proto := compileLua(t, "local n = 0 while true do n = n + 1 end", "=terminated")
	done := make(chan error, 1)
	go func() {
		_, err := co.Run(proto)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "execution interrupted") {
			t.Fatalf("coroutine of a terminated VM ran to completion: err=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("coroutine of a terminated VM never unwound")
	}
}

// TestVMCloseRunsReapedHandlersOneAtATime checks that VM.Close does not set two
// abandoned coroutines' <close> handlers running at the same time. Each handler
// runs Lua on its own coroutine's goroutine, so overlapping them would put two
// goroutines into one VM family's state — and giving up on a slow handler is
// exactly what invites the overlap.
func TestVMCloseRunsReapedHandlersOneAtATime(t *testing.T) {
	var mu sync.Mutex
	var order []int64 // one entry per switch between coroutines
	var beats int64

	v := vm.New(vm.WithContext(context.Background()))
	stdlib.Open(v)
	v.SetGlobal("beat", vm.NewNativeFunc(func(cv *vm.VM) int {
		k, _ := cv.Get(1).ToInt()
		mu.Lock()
		if len(order) == 0 || order[len(order)-1] != k {
			order = append(order, k)
		}
		beats++
		mu.Unlock()
		return 0
	}))

	proto := compileLua(t, `
		for k = 1, 2 do
			local co = coroutine.create(function()
				local x <close> = setmetatable({}, {__close = function()
					for i = 1, 50000000 do beat(k) end
				end})
				coroutine.yield()
			end)
			coroutine.resume(co)
		end
	`, "=reaporder")
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("run: %v", err)
	}

	// A deadline short enough that neither handler can finish inside it, so
	// Close has to stop waiting for the first one before the second is woken.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := v.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if beats == 0 {
		t.Fatalf("test did not exercise the reap path: no <close> handler ran")
	}
	seen := map[int64]bool{}
	for _, k := range order {
		if seen[k] {
			head := order
			if len(head) > 20 {
				head = head[:20]
			}
			t.Fatalf("VM.Close ran reaped <close> handlers concurrently: coroutine "+
				"switching went %v (%d switches; an id that comes back has been "+
				"interleaved with another handler)", head, len(order))
		}
		seen[k] = true
	}
}

// TestVMCloseReturnsWithBlockingCloseHandler checks that VM.Close returns even
// when an abandoned coroutine's <close> handler does not, and even when the
// caller's context has no deadline of its own — context.Background() is the
// documented `defer v.Close(ctx)` lifecycle, and a receive on its nil Done
// channel is not a bound.
func TestVMCloseReturnsWithBlockingCloseHandler(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)
	proto := compileLua(t, `
		local co = coroutine.create(function()
			local x <close> = setmetatable({}, {__close = function() while true do end end})
			coroutine.yield()
		end)
		coroutine.resume(co)
	`, "=closehang")
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("run: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- v.Close(context.Background()) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatalf("VM.Close(context.Background()) never returned: an abandoned " +
			"coroutine's <close> handler blocked it")
	}
}
