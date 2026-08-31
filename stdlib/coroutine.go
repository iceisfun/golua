package stdlib

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/iceisfun/golua/v2/vm"
)

// coroutineStopGrace bounds every wait in which one goroutine has to see
// another goroutine of the same VM family stop running Lua.
//
// A coroutine runs on its own goroutine but against the family's shared state
// (globals, tables, the string cache), so whoever abandons a *running*
// coroutine — a resume that gave up on a cancelled context, or VM.Close reaping
// a suspended one — must wait for it to stop first. Those waits terminate on
// their own: the abandoning side has already made the coroutine's VM unwind at
// its next interrupt poll (a cancelled context, or vm.VM.Terminate).
//
// The grace is the backstop for a coroutine that does not reach a poll. In
// practice that is one parked inside a native call which observes neither
// signal — a blocking read, say — and such a coroutine is touching none of the
// shared state these waits protect. It is not the only way, though: the polls
// sit at loop back-edges and at calls, so a long enough run of straight-line
// Lua can outlast the grace as well. A second's worth of it is a great deal of
// Lua, but the guarantee is a bound on the wait, not a proof that the coroutine
// has stopped.
const coroutineStopGrace = time.Second

// coArgType returns the type name for error messages, using "no value" when argc is insufficient.
func coArgType(v *vm.VM, idx int) string {
	if v.ArgCount() < idx {
		return "no value"
	}
	return v.Get(idx).Type()
}

// ctxDone returns the Done channel from the VM's context, or nil if no context is set.
// A nil channel blocks forever in select, which is the correct fallback.
func ctxDone(v *vm.VM) <-chan struct{} {
	if ctx := v.Context(); ctx != nil {
		return ctx.Done()
	}
	return nil
}

func getThreadTable(v *vm.VM, idx int, funcName string) *vm.Table {
	arg := v.Get(idx)
	if !arg.IsTable() {
		callerArgError(v, idx, funcName, fmt.Sprintf("thread expected, got %s", coArgType(v, idx)))
	}
	tbl, _ := arg.AsTable().(*vm.Table)
	if tbl == nil || !tbl.IsThread() {
		callerArgError(v, idx, funcName, fmt.Sprintf("thread expected, got %s", coArgType(v, idx)))
	}
	return tbl
}

// coroutineStatus represents the lifecycle state of a coroutine.
type coroutineStatus string

const (
	statusSuspended coroutineStatus = "suspended"
	statusRunning   coroutineStatus = "running"
	statusDead      coroutineStatus = "dead"
	statusNormal    coroutineStatus = "normal"
)

// Cached status result Values. coroutine.status is called in tight polling
// loops (`while coroutine.status(co) ~= "dead"`), and storing a string into the
// Value.ptr interface heap-allocates, so build each of the four status strings
// once instead of per call. Values are immutable, so sharing is safe.
var (
	statusValSuspended = vm.NewString(string(statusSuspended))
	statusValRunning   = vm.NewString(string(statusRunning))
	statusValDead      = vm.NewString(string(statusDead))
	statusValNormal    = vm.NewString(string(statusNormal))
)

// coIDKey is the cached "__coroutine_id" key Value, for the few lookups that go
// through the LuaTable interface (which has no boxing-free GetString).
var coIDKey = vm.NewString("__coroutine_id")

// statusValue maps a coroutineStatus to its cached Value.
func statusValue(s coroutineStatus) vm.Value {
	switch s {
	case statusRunning:
		return statusValRunning
	case statusDead:
		return statusValDead
	case statusNormal:
		return statusValNormal
	default:
		return statusValSuspended
	}
}

// Coroutine represents a Lua coroutine
type Coroutine struct {
	id             int
	fn             vm.Value        // The function to run
	status         coroutineStatus // lifecycle state
	started        bool            // Whether the goroutine has been started
	resumeChClosed bool            // Whether resumeCh has been closed by coClose
	reapAbandoned  bool            // VM.Close marked it dead without its goroutine finishing
	vm             *vm.VM          // Reference to the VM
	coVM           *vm.VM          // The coroutine's own VM (set after first resume)
	thread         vm.Value        // Thread object (table) for coroutine.running
	resumeCh       chan []vm.Value // Channel to send resume args
	yieldCh        chan []vm.Value // Channel to receive yield values
	doneCh         chan struct{}   // Channel to signal completion
	result         []vm.Value      // Final return values
	err            error           // Error if panicked
	mu             sync.Mutex
}

// coRegistry is the per-VM coroutine registry, stored in VM internal state.
type coRegistry struct {
	mu         sync.Mutex
	nextID     int
	coroutines map[int]*Coroutine
}

const coRegistryKey = "coroutine"

// getCoRegistry returns the per-VM coroutine registry, creating it lazily.
func getCoRegistry(v *vm.VM) *coRegistry {
	if r := v.InternalState(coRegistryKey); r != nil {
		return r.(*coRegistry)
	}
	reg := &coRegistry{
		coroutines: make(map[int]*Coroutine),
	}
	v.SetInternalState(coRegistryKey, reg)
	return reg
}

func openCoroutine(v *vm.VM) {
	co := vm.NewEmptyTable()

	co.SetString("create", vm.NewNativeFunc(coCreate))
	co.SetString("resume", vm.NewNativeFunc(coResume))
	co.SetString("yield", vm.NewNativeFunc(coYield))
	co.SetString("status", vm.NewNativeFunc(coStatus))
	co.SetString("running", vm.NewNativeFunc(coRunning))
	co.SetString("wrap", vm.NewNativeFunc(coWrap))
	co.SetString("isyieldable", vm.NewNativeFunc(coIsYieldable))
	co.SetString("close", vm.NewNativeFunc(coClose))

	v.SetGlobal("coroutine", vm.NewTable(co))

	// Create and store the main thread object so coroutine.running()
	// returns a stable, non-nil identity for the main thread.
	mainThread := vm.NewEmptyTable()
	mainThread.SetString("__coroutine_id", vm.NewInt(0))
	mainThread.SetThread(true)
	mainThread.SetVMRef(v)
	v.SetThreadObj(vm.NewTable(mainThread))

	// On VM.Close, reap the goroutines of any still-suspended coroutines so an
	// abandoned suspended coroutine (never run to completion, never
	// coroutine.close()'d) doesn't leak its goroutine past the VM's lifetime.
	// See reapCoroutines and wontfix/coroutine-goroutine-leak.
	v.OnClose(func(ctx context.Context) { reapCoroutines(v, ctx) })
}

// reapCoroutines force-terminates every still-suspended coroutine tracked by the
// VM's registry, so their backing goroutines exit. Registered as a VM close hook
// (runs on VM.Close). Go cannot reap a parked goroutine, but closing its resume
// channel unblocks it; the goroutine then runs its <close> handlers and returns
// — the same mechanism coroutine.close() uses. This bounds the
// coroutine-goroutine leak to the VM's lifetime: create a VM, use it, Close it,
// and abandoned suspended coroutines are reclaimed.
func reapCoroutines(v *vm.VM, ctx context.Context) {
	reg := getCoRegistry(v)
	reg.mu.Lock()
	pending := make([]*Coroutine, 0, len(reg.coroutines))
	for _, co := range reg.coroutines {
		pending = append(pending, co)
	}
	reg.coroutines = make(map[int]*Coroutine)
	reg.mu.Unlock()
	if len(pending) == 0 {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}
	// Each coroutine gets its own grace rather than the sweep sharing one, so
	// that a handler which uses up its whole allowance cannot deny the grace to
	// the coroutines reaped after it. A shared deadline made the first slow
	// handler silently skip every remaining coroutine's <close>.
	//
	// The caller's context bounds the sweep, but only if it was still live when
	// the sweep began. Close is documented to be called as
	// `defer v.Close(context.Background())`, and a caller who hands over a
	// context that has already expired is asking to be unblocked, not asking to
	// skip the handlers altogether: an expired context used as the grace closes
	// its Done channel immediately, which would terminate every coroutine
	// before its handler had run at all.
	var sweepDone <-chan struct{}
	if ctx.Err() == nil {
		sweepDone = ctx.Done()
	}

	for _, co := range pending {
		if !reapCoroutine(co, sweepDone) {
			// This coroutine's goroutine is still running Lua against the
			// family's shared state and did not stop when asked. Waking the
			// next coroutine now would set a second goroutine running alongside
			// it, so end the sweep and leave the rest suspended where they are.
			//
			// That costs more than the parked goroutines themselves (see
			// wontfix/coroutine-goroutine-leak): the <close> handlers of the
			// coroutines this sweep had not reached yet do not run at all. Two
			// goroutines writing one VM family's tables is the worse outcome of
			// the two, and reaching this point at all means a handler has
			// already ignored both a full grace and an explicit termination.
			return
		}
	}
}

// reapCoroutine terminates one suspended coroutine's goroutine. It mirrors the
// suspended-close path of coClose (close resumeCh -> the goroutine unwinds via
// CloseAllTBC and returns), but: (1) only acts on started, suspended coroutines
// with an un-closed resume channel; (2) recovers per-coroutine so one handler's
// error doesn't abort the sweep; (3) bounds the wait, in two stages, so a
// blocking <close> handler can neither hang VM.Close nor outlive it.
//
// The grace is when this coroutine's handlers stop being given the benefit of
// the doubt, and the hard bound is the end of the wait entirely; both are per
// coroutine. Between them the coroutine's VM is told to unwind, which is what
// makes the second wait terminate rather than merely expire. sweepDone, when
// non-nil, is the caller's context: it can end the grace early but never
// replaces it.
//
// Returns whether the coroutine's goroutine actually stopped. A false return
// means it is still executing Lua against state this VM family shares, so the
// caller must not start another one alongside it.
func reapCoroutine(co *Coroutine, sweepDone <-chan struct{}) bool {
	defer func() { _ = recover() }()

	co.mu.Lock()
	reap := co.started && co.status == statusSuspended && !co.resumeChClosed
	if reap {
		co.status = statusRunning
		co.resumeChClosed = true
	}
	coVM := co.coVM
	co.mu.Unlock()
	if !reap {
		return true
	}

	close(co.resumeCh)

	soft := time.NewTimer(coroutineStopGrace)
	defer soft.Stop()
	hard := time.NewTimer(2 * coroutineStopGrace)
	defer hard.Stop()

	finished := false
	graceOver := false
	select {
	case <-co.doneCh:
		finished = true
	case <-soft.C:
		graceOver = true
	case <-sweepDone:
		// nil unless the caller's context was live when the sweep began, in
		// which case a select on it never fires.
		graceOver = true
	}

	if graceOver {
		// The grace ran out with the <close> handlers still running Lua on the
		// coroutine's own goroutine. Returning here would leave that goroutine
		// writing to the globals and tables the closing VM shares with it, so
		// ask its VM to unwind at the next interrupt poll and wait for the
		// goroutine to actually finish.
		if coVM != nil {
			coVM.Terminate()
		}
		select {
		case <-co.doneCh:
			finished = true
		case <-hard.C:
		}
	}

	co.mu.Lock()
	co.status = statusDead
	// This is the one place statusDead is published while the goroutine may
	// still be running (and still owns its coroutine VM). Record that, so a
	// later coroutine.close neither waits forever on a doneCh that may never
	// close nor touches a VM another goroutine is still using.
	co.reapAbandoned = !finished
	co.mu.Unlock()
	return finished
}

// coroutine.create(f) -> thread
func coCreate(v *vm.VM) int {
	fn := v.Get(1)
	if !fn.IsFunction() && !fn.IsNativeFunc() {
		callerArgError(v, 1, "coroutine.create", fmt.Sprintf("function expected, got %s", coArgType(v, 1)))
	}

	reg := getCoRegistry(v)
	reg.mu.Lock()
	reg.nextID++
	id := reg.nextID
	reg.mu.Unlock()

	co := &Coroutine{
		id:       id,
		fn:       fn,
		status:   statusSuspended,
		vm:       v,
		resumeCh: make(chan []vm.Value, 1),
		yieldCh:  make(chan []vm.Value, 1),
		doneCh:   make(chan struct{}),
	}

	reg.mu.Lock()
	reg.coroutines[id] = co
	reg.mu.Unlock()

	// Create a table to represent the coroutine (with metatable for type identification)
	coTable := vm.NewEmptyTable()
	coTable.SetString("__coroutine_id", vm.NewInt(int64(id)))
	coTable.SetThread(true)

	threadVal := vm.NewTable(coTable)
	co.thread = threadVal

	// Create the coroutine VM eagerly so that debug.sethook/gethook can
	// operate on it before the first resume (matching Lua 5.4 behavior).
	coVM := vm.NewCoroutineVM(v, co.yieldCh, co.resumeCh, id)
	coVM.SetThreadObj(threadVal)
	co.coVM = coVM
	coTable.SetVMRef(coVM)

	v.Set(0, threadVal)
	return 1
}

// coroutine.resume(co [, val1, ...]) -> ok, results...
func coResume(v *vm.VM) int {
	coTable := getThreadTable(v, 1, "coroutine.resume")
	idVal := coTable.GetString("__coroutine_id")
	if idVal.IsNil() {
		callerArgError(v, 1, "coroutine.resume", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	id, _ := idVal.ToInt()

	// The main thread (id=0) is never in the coroutines map.
	// It is always "running" or "normal", never "suspended", so resume fails.
	if int(id) == 0 {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("cannot resume non-suspended coroutine"))
		return 2
	}

	reg := getCoRegistry(v)
	reg.mu.Lock()
	co := reg.coroutines[int(id)]
	reg.mu.Unlock()

	if co == nil {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("cannot resume dead coroutine"))
		return 2
	}

	co.mu.Lock()
	status := co.status
	co.mu.Unlock()

	if status != statusSuspended {
		var msg string
		if status == statusDead {
			msg = "cannot resume dead coroutine"
		} else {
			msg = "cannot resume non-suspended coroutine"
		}
		v.Set(0, vm.False)
		v.Set(1, vm.NewString(msg))
		return 2
	}

	// Collect arguments
	argc := v.ArgCount()
	args := make([]vm.Value, argc-1)
	for i := 2; i <= argc; i++ {
		args[i-2] = v.Get(i)
	}

	// Set caller's status to normal while the resumed coroutine runs
	callerID := v.CoroutineID()
	if callerID != 0 {
		reg.mu.Lock()
		if caller := reg.coroutines[callerID]; caller != nil {
			caller.mu.Lock()
			caller.status = statusNormal
			caller.mu.Unlock()
		}
		reg.mu.Unlock()
	}

	// restoreCallerStatus restores the caller's coroutine status to running
	restoreCallerStatus := func() {
		if callerID != 0 {
			reg.mu.Lock()
			if caller := reg.coroutines[callerID]; caller != nil {
				caller.mu.Lock()
				caller.status = statusRunning
				caller.mu.Unlock()
			}
			reg.mu.Unlock()
		}
	}

	// Start the goroutine and send args under the same lock to prevent
	// a concurrent coClose from closing resumeCh between the status
	// transition and the send. The send is non-blocking because the
	// channel is buffered(1) and empty when the coroutine is suspended.
	co.mu.Lock()
	if co.resumeChClosed {
		co.mu.Unlock()
		restoreCallerStatus()
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("cannot resume dead coroutine"))
		return 2
	}
	if !co.started {
		co.started = true
		co.status = statusRunning
		go runCoroutine(co)
	} else {
		co.status = statusRunning
	}
	// Publish our instruction consumption before the coroutine picks up the
	// shared budget, so the limit bounds the family and not each VM apart.
	v.SyncInstructionBudget()
	co.resumeCh <- args
	co.mu.Unlock()

	// Adopt whatever the coroutine charged to the budget before the resumer's
	// bytecode runs again. Deferred so it also covers the abandoned-coroutine
	// (context cancelled) branch and the re-panicked os.exit sentinel.
	defer v.SyncInstructionBudget()

	// Wait for yield or completion
	select {
	case results := <-co.yieldCh:
		restoreCallerStatus()
		needed := v.Base() + 1 + len(results)
		if !v.CheckStack(needed) {
			v.Set(0, vm.False)
			v.Set(1, vm.NewString("C stack overflow"))
			return 2
		}
		v.EnsureStack(needed)
		v.Set(0, vm.True)
		for i, r := range results {
			v.Set(i+1, r)
		}
		return 1 + len(results)
	case <-co.doneCh:
		restoreCallerStatus()
		co.mu.Lock()
		err := co.err
		result := co.result
		co.mu.Unlock()

		if err != nil {
			// os.exit inside the coroutine: the exit sentinel is not a
			// coroutine error — re-panic it on the resumer's goroutine so it
			// propagates through the resumer's ProtectedCall chain (and any
			// enclosing coroutines) exactly as it does through pcall.
			if exitErr, ok := err.(*vm.LuaExitError); ok {
				panic(exitErr)
			}
			v.Set(0, vm.False)
			// Preserve the original Lua error value if available
			var errVal vm.Value
			if le, ok := err.(*vm.LuaError); ok {
				errVal = le.Value
			} else {
				errVal = vm.NewString(err.Error())
			}
			// Lua 5.5: nil error object is replaced by "<no error object>".
			// Apply symmetrically with pcall/xpcall on the resume path.
			if errVal.IsNil() {
				errVal = vm.NewString("<no error object>")
			}
			v.Set(1, errVal)
			return 2
		}

		needed := v.Base() + 1 + len(result)
		if !v.CheckStack(needed) {
			v.Set(0, vm.False)
			v.Set(1, vm.NewString("C stack overflow"))
			return 2
		}
		v.EnsureStack(needed)
		v.Set(0, vm.True)
		for i, r := range result {
			v.Set(i+1, r)
		}
		return 1 + len(result)
	case <-ctxDone(v):
		restoreCallerStatus()
		// The resume is over, but the coroutine's goroutine is not: let it come
		// to a stop before the resumer touches the family's shared state again.
		awaitCoroutineStop(co)
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("execution interrupted: "+v.Context().Err().Error()))
		return 2
	}
}

// awaitCoroutineStop waits for a coroutine the resumer is about to abandon to
// stop running: it either finishes (doneCh) or yields and parks on its resume
// channel (yieldCh). Until one of those happens its goroutine is still
// executing Lua against the globals, tables and string cache the resumer shares
// with it, and returning to the resumer's own bytecode would put two goroutines
// into one VM family's state at once.
//
// Every caller reaches here because the family's context has been cancelled,
// which the coroutine's VM observes at its own next interrupt poll, so the wait
// normally ends within a few instructions. coroutineStopGrace is the backstop.
func awaitCoroutineStop(co *Coroutine) {
	timer := time.NewTimer(coroutineStopGrace)
	defer timer.Stop()
	select {
	case <-co.doneCh:
	case <-co.yieldCh:
		// It yielded on the way out: parked on resumeCh, not running, which is
		// all this wait needs. Taking the value off the channel is what unblocks
		// it, so this receive is not just an observation — the yielded values
		// are consumed and dropped here, together with the abandoned resume that
		// was the only thing that would ever have returned them.
	case <-timer.C:
	}
}

// runCoroutine runs the coroutine function in a goroutine
func runCoroutine(co *Coroutine) {
	defer func() {
		// Publish this coroutine's instruction consumption into the budget
		// shared with the resumer before doneCh is closed, so the resumer
		// adopts it when it wakes up.
		co.mu.Lock()
		coVM := co.coVM
		co.mu.Unlock()
		if coVM != nil {
			coVM.SyncInstructionBudget()
		}
		if r := recover(); r != nil {
			// Lua 5.5: coroutine.close(coroutine.running()) terminates the
			// coroutine cleanly via the vm.CoroutineSelfClose sentinel. Treat
			// this as a normal completion, not an error.
			if sc, ok := r.(vm.CoroutineSelfClose); ok {
				co.mu.Lock()
				// A <close> handler that errored during self-close terminates the
				// coroutine with that error: resume reports (false, errval).
				if sc.HasErr {
					co.err = sc.Err
				}
				co.status = statusDead
				co.mu.Unlock()
				close(co.doneCh)
				return
			}
			co.mu.Lock()
			// Preserve *LuaError so resume can return the original Lua value
			if le, ok := r.(*vm.LuaError); ok {
				co.err = le
			} else if err, ok := r.(error); ok {
				co.err = err
			} else {
				co.err = fmt.Errorf("%v", r)
			}
			co.status = statusDead
			co.mu.Unlock()
		}
		close(co.doneCh)
	}()

	co.mu.Lock()
	co.status = statusRunning
	co.mu.Unlock()

	// Wait for first resume args
	args, ok := <-co.resumeCh
	if !ok {
		// Channel closed by coClose before first resume completed
		co.mu.Lock()
		co.status = statusDead
		co.mu.Unlock()
		return
	}

	// Call the function
	var results []vm.Value
	var err error

	// Use the coroutine VM created eagerly in coCreate
	co.mu.Lock()
	coVM := co.coVM
	co.mu.Unlock()

	// Take over the family's instruction budget: whatever the resumer (and any
	// coroutine before us) consumed counts against our own limit check.
	if coVM != nil {
		coVM.SyncInstructionBudget()
	}

	if co.fn.IsFunction() {
		results, err = coVM.CallCoroutine(co.fn.AsClosure(), args)
	} else if co.fn.IsNativeFunc() {
		// Use coroutine VM so yield works from functions called by the native func
		results, err = coVM.ProtectedCallCoroutine(co.fn, args)
	}

	// On NORMAL completion, defensively copy the return values out of any VM
	// buffer they may alias (OP_RETURN returns a slice into vm.retBuf), then
	// release that buffer so locals from the completed frames become collectable
	// (reference Lua frees a dead coroutine's stack). Without the copy, clearing
	// retBuf would also wipe co.result. When the coroutine died via an ERROR, the
	// suspended frame at the error point is retained for post-mortem debugging
	// (debug.getinfo/getlocal/setlocal on the dead coroutine), so leave it intact.
	if err == nil && len(results) > 0 {
		safe := make([]vm.Value, len(results))
		copy(safe, results)
		results = safe
	}

	co.mu.Lock()
	co.result = results
	if err != nil {
		co.err = err
	}
	co.status = statusDead
	co.mu.Unlock()

	if err == nil {
		coVM.ReleaseDeadStack()
	}
}

// coroutine.yield(...) -> resume args
func coYield(v *vm.VM) int {
	// This is called from within a coroutine
	// We need to send values to the yieldCh and wait for resumeCh

	// Collect values to yield
	argc := v.ArgCount()
	values := make([]vm.Value, argc)
	for i := 1; i <= argc; i++ {
		values[i-1] = v.Get(i)
	}

	// Signal yield
	yieldCh, resumeCh := v.GetCoroutineChannels()
	if yieldCh == nil || resumeCh == nil {
		// Use LuaError to avoid addCallerLocation prefix — Lua 5.4 raises
		// this from C code so no file:line is prepended.
		panic(&vm.LuaError{Value: vm.NewString("attempt to yield from outside a coroutine")})
	}
	if !v.IsYieldableContext() {
		// Use LuaError to avoid addCallerLocation prefix — Lua 5.4 raises
		// this from C code (lua_yieldk) so no file:line is prepended.
		panic(&vm.LuaError{Value: vm.NewString("attempt to yield across a C-call boundary")})
	}

	// Mark coroutine as suspended before yielding
	coID := v.CoroutineID()
	reg := getCoRegistry(v)
	if coID != 0 {
		reg.mu.Lock()
		if co := reg.coroutines[coID]; co != nil {
			co.mu.Lock()
			co.status = statusSuspended
			co.mu.Unlock()
		}
		reg.mu.Unlock()
	}

	// Hand the shared instruction budget back to the resumer along with the
	// yielded values; pick it up again below when we are resumed.
	v.SyncInstructionBudget()

	yieldCh <- values

	// Wait for resume - mark as running when we get it
	args, ok := <-resumeCh
	if !ok {
		// Channel closed by coClose — close TBC vars, then terminate.
		// Set closingCoroutine so __close errors propagate through any
		// pcall/xpcall boundaries in the coroutine's call stack.
		v.EnterClosingCoroutine()
		v.CloseAllTBC()
		runtime.Goexit()
	}

	// Resumed: take the shared budget back.
	v.SyncInstructionBudget()

	// Check for context cancellation after waking
	if ctx := v.Context(); ctx != nil {
		if err := ctx.Err(); err != nil {
			panic(fmt.Sprintf("execution interrupted: %v", err))
		}
	}

	if coID != 0 {
		reg.mu.Lock()
		if co := reg.coroutines[coID]; co != nil {
			co.mu.Lock()
			co.status = statusRunning
			co.mu.Unlock()
		}
		reg.mu.Unlock()
	}

	// Return the resume args
	v.EnsureStack(v.Base() + len(args))
	for i, arg := range args {
		v.Set(i, arg)
	}
	return len(args)
}

// coroutine.status(co) -> string
func coStatus(v *vm.VM) int {
	coTable := getThreadTable(v, 1, "coroutine.status")
	idVal := coTable.GetString("__coroutine_id")
	if idVal.IsNil() {
		callerArgError(v, 1, "coroutine.status", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	id, _ := idVal.ToInt()

	// The main thread (id=0) is not in the coroutines map.
	// Its status depends on whether the caller is the main thread itself.
	if int(id) == 0 {
		if v.CoroutineID() == 0 {
			v.Set(0, statusValRunning)
		} else {
			// Main thread is suspended while a coroutine runs
			v.Set(0, statusValNormal)
		}
		return 1
	}

	reg := getCoRegistry(v)
	reg.mu.Lock()
	co := reg.coroutines[int(id)]
	reg.mu.Unlock()

	if co == nil {
		v.Set(0, statusValDead)
		return 1
	}

	co.mu.Lock()
	status := co.status
	co.mu.Unlock()

	v.Set(0, statusValue(status))
	return 1
}

// coroutine.running() -> thread, boolean
func coRunning(v *vm.VM) int {
	v.Set(0, v.ThreadObj())
	v.Set(1, vm.NewBool(v.CoroutineID() == 0))
	return 2
}

// coroutine.wrap(f) -> function
func coWrap(v *vm.VM) int {
	fn := v.Get(1)
	if !fn.IsFunction() && !fn.IsNativeFunc() {
		callerArgError(v, 1, "coroutine.wrap", fmt.Sprintf("function expected, got %s", coArgType(v, 1)))
	}

	// Create the coroutine
	reg := getCoRegistry(v)
	reg.mu.Lock()
	reg.nextID++
	id := reg.nextID
	reg.mu.Unlock()

	// Create a thread table for this coroutine (for coroutine.running inside it)
	coTable := vm.NewEmptyTable()
	coTable.SetString("__coroutine_id", vm.NewInt(int64(id)))
	coTable.SetThread(true)

	threadVal := vm.NewTable(coTable)

	co := &Coroutine{
		id:       id,
		fn:       fn,
		status:   statusSuspended,
		vm:       v,
		thread:   threadVal,
		resumeCh: make(chan []vm.Value, 1),
		yieldCh:  make(chan []vm.Value, 1),
		doneCh:   make(chan struct{}),
	}

	// Create the coroutine VM eagerly (matching coCreate)
	coVM := vm.NewCoroutineVM(v, co.yieldCh, co.resumeCh, id)
	coVM.SetThreadObj(threadVal)
	co.coVM = coVM
	coTable.SetVMRef(coVM)

	reg.mu.Lock()
	reg.coroutines[id] = co
	reg.mu.Unlock()

	// Return a wrapper function that resumes the coroutine on each call.
	// Unlike coroutine.resume, errors are raised directly (not returned as false, err).
	wrapper := vm.NewNativeFuncWithNups(func(v *vm.VM) int {
		co.mu.Lock()
		status := co.status
		co.mu.Unlock()

		if status != statusSuspended {
			if status == statusDead {
				panic("cannot resume dead coroutine")
			}
			panic("cannot resume non-suspended coroutine")
		}

		// Collect arguments
		argc := v.ArgCount()
		args := make([]vm.Value, argc)
		for i := 1; i <= argc; i++ {
			args[i-1] = v.Get(i)
		}

		// Set caller's status to normal while the resumed coroutine runs
		callerID := v.CoroutineID()
		if callerID != 0 {
			reg.mu.Lock()
			if caller := reg.coroutines[callerID]; caller != nil {
				caller.mu.Lock()
				caller.status = statusNormal
				caller.mu.Unlock()
			}
			reg.mu.Unlock()
		}
		restoreCallerStatus := func() {
			if callerID != 0 {
				reg.mu.Lock()
				if caller := reg.coroutines[callerID]; caller != nil {
					caller.mu.Lock()
					caller.status = statusRunning
					caller.mu.Unlock()
				}
				reg.mu.Unlock()
			}
		}

		// Start the goroutine and send args under the same lock.
		co.mu.Lock()
		if co.resumeChClosed {
			co.mu.Unlock()
			restoreCallerStatus()
			panic("cannot resume dead coroutine")
		}
		if !co.started {
			co.started = true
			co.status = statusRunning
			go runCoroutine(co)
		} else {
			co.status = statusRunning
		}
		// Publish our instruction consumption before the coroutine picks up
		// the shared budget (see coResume).
		v.SyncInstructionBudget()
		co.resumeCh <- args
		co.mu.Unlock()

		// Adopt whatever the coroutine charged to the budget, including on the
		// abandoned-coroutine (context cancelled) and re-raise branches.
		defer v.SyncInstructionBudget()

		// Wait for yield or completion
		select {
		case results := <-co.yieldCh:
			restoreCallerStatus()
			needed := v.Base() + len(results)
			if !v.CheckStack(needed) {
				panic("C stack overflow")
			}
			v.EnsureStack(needed)
			for i, r := range results {
				v.Set(i, r)
			}
			return len(results)
		case <-co.doneCh:
			restoreCallerStatus()
			co.mu.Lock()
			err := co.err
			result := co.result
			coVM := co.coVM
			co.mu.Unlock()

			if err != nil {
				// os.exit inside the coroutine: propagate the exit sentinel
				// unchanged (reference exit() terminates without unwinding,
				// so no TBC close pass here).
				if exitErr, ok := err.(*vm.LuaExitError); ok {
					panic(exitErr)
				}
				// Close TBC vars on the coroutine VM before re-raising.
				// Lua 5.4: wrap closes TBC vars when the error propagates.
				// If a __close handler errors, that error replaces the original.
				if coVM != nil {
					var errVal vm.Value
					if le, ok := err.(*vm.LuaError); ok {
						errVal = le.Value
					} else {
						errVal = vm.NewString(err.Error())
					}
					if closeErr := coVM.ClosePendingTBC(errVal, true); closeErr != nil {
						err = closeErr
					}
					// Those handlers ran on this goroutine but against the dead
					// coroutine's VM, so their work is charged there: publish it
					// (the deferred sync above then adopts it) or a script could
					// buy unbounded __close time by erroring coroutine after
					// coroutine.
					coVM.SyncInstructionBudget()
				}
				// coroutine.wrap adds the caller location to string errors
				// via luaL_where(L,1) before re-raising (Lua's luaB_auxwrap).
				// This prepend is UNCONDITIONAL — if the inner error already
				// begins with the same source:line: (its origin line equals the
				// wrap call site), the prefix legitimately appears twice. Wrap
				// the result as *LuaError so the VM panic handler does not run
				// its deduping AddCallerLocation over it again. Non-string
				// errors (tables, etc.) propagate unchanged.
				if le, ok := err.(*vm.LuaError); ok {
					if le.Value.IsString() {
						msg := v.PrependCallerLocation(le.Value.AsString())
						panic(&vm.LuaError{Value: vm.NewString(msg)})
					}
					// Lua 5.5: a nil error object is converted to the string
					// "<no error object>" at throw time (luaG_errormsg), so by
					// the time luaB_auxwrap re-raises it is a string and gets the
					// caller location prepended just like any other string error.
					if le.Value.IsNil() {
						msg := v.PrependCallerLocation("<no error object>")
						panic(&vm.LuaError{Value: vm.NewString(msg)})
					}
					panic(le)
				}
				panic(&vm.LuaError{Value: vm.NewString(v.PrependCallerLocation(err.Error()))})
			}

			needed := v.Base() + len(result)
			if !v.CheckStack(needed) {
				panic("C stack overflow")
			}
			v.EnsureStack(needed)
			for i, r := range result {
				v.Set(i, r)
			}
			return len(result)
		case <-ctxDone(v):
			restoreCallerStatus()
			// See coResume: the coroutine's goroutine has to stop before this
			// one goes back to the family's shared state.
			awaitCoroutineStop(co)
			panic("execution interrupted: " + v.Context().Err().Error())
		}
	}, 1) // 1 upvalue: the wrapped coroutine thread
	wrapper.SetNativeFuncUpvalue(1, vm.NewTable(coTable))

	v.Set(0, wrapper)
	return 1
}

// coroutine.close(co) -> ok [, errmsg]
// Lua 5.5: if no argument is given, defaults to the current thread. Closing
// the main thread always errors with "cannot close main thread".
func coClose(v *vm.VM) int {
	var coTable *vm.Table
	if v.ArgCount() < 1 {
		// Default to the current thread (matches Lua 5.5 getoptco behavior).
		thread := v.ThreadObj()
		if tbl, ok := thread.AsTable().(*vm.Table); ok && tbl != nil && tbl.IsThread() {
			coTable = tbl
		} else {
			callerArgError(v, 1, "coroutine.close", "thread expected, got no value")
		}
	} else {
		coTable = getThreadTable(v, 1, "coroutine.close")
	}

	idVal := coTable.GetString("__coroutine_id")
	if idVal.IsNil() {
		callerArgError(v, 1, "coroutine.close", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	id, _ := idVal.ToInt()

	// The main thread has id=0. Closing it (whether passed explicitly or via
	// the no-arg default when running on main) is always an error per 5.5.
	if int(id) == 0 {
		// When called from a coroutine, the main thread is in "normal" status
		// and "cannot close a normal coroutine" would be more precise, but
		// Lua 5.5's dispatch order checks the main-thread identity only on the
		// RUNNING branch. In practice, the main thread is either currently
		// running (when id==v.CoroutineID()==0) or is the parent of a running
		// coroutine (status == normal). 5.5's error for the latter is still
		// "cannot close a normal coroutine", but our only invariant is that
		// the main thread itself can never be closed: report the specific
		// 5.5 message when we are running on main.
		if v.CoroutineID() == 0 {
			panic("cannot close main thread")
		}
		panic("cannot close a normal coroutine")
	}

	// Lua 5.5: coroutine.close(coroutine.running()) terminates the running
	// coroutine cleanly. Run pending <close> handlers on the current call
	// stack, then long-jump out via vm.CoroutineSelfClose. The recover in
	// runCoroutine treats this sentinel as a normal completion and the
	// resumer sees (true, nil).
	if int(id) == v.CoroutineID() {
		// Run pending <close> handlers on the current call stack. If one errors,
		// the running coroutine still terminates (self-close), but the error is
		// carried on the sentinel so the resumer observes (false, errval) rather
		// than (true) — matching Lua 5.5 lua_closethread semantics.
		selfClose := vm.CoroutineSelfClose{}
		func() {
			defer func() {
				if r := recover(); r != nil {
					selfClose.HasErr = true
					if le, ok := r.(*vm.LuaError); ok {
						selfClose.Err = le
					} else if err, ok := r.(error); ok {
						selfClose.Err = err
					} else {
						selfClose.Err = fmt.Errorf("%v", r)
					}
				}
			}()
			v.CloseAllTBC()
		}()
		panic(selfClose)
	}

	reg := getCoRegistry(v)
	reg.mu.Lock()
	co := reg.coroutines[int(id)]
	reg.mu.Unlock()

	if co == nil {
		// Already dead/collected — that's fine
		v.Set(0, vm.True)
		return 1
	}

	co.mu.Lock()
	status := co.status
	co.mu.Unlock()

	if status == statusRunning {
		panic("cannot close a running coroutine")
	}

	if status == statusNormal {
		panic("cannot close a normal coroutine")
	}

	if status == statusDead {
		reg.mu.Lock()
		delete(reg.coroutines, int(id))
		reg.mu.Unlock()

		// Observing status=dead is NOT enough to take over the coroutine's VM:
		// runCoroutine publishes statusDead under co.mu and only afterwards
		// finishes with the VM (its stack release and its final budget sync,
		// see the deferred SyncInstructionBudget above). doneCh is closed last,
		// so waiting on it is what actually orders those writes before the
		// reads below — otherwise a resume that gave up on a cancelled context
		// leaves this goroutine racing the dying coroutine for coVM.
		//
		// The wait is bounded. Every path that publishes statusDead from the
		// coroutine goroutine reaches runCoroutine's outermost defer (which
		// closes doneCh) without blocking again, and a closer that already
		// drained the goroutine below closed doneCh before setting statusDead.
		// !started means no goroutine was ever launched, so nothing will close
		// doneCh — and equally nothing else can be touching coVM. reapAbandoned
		// is the single case where statusDead was published without the
		// goroutine finishing (VM.Close hit its deadline waiting for a <close>
		// handler): that goroutine still owns coVM and may never close doneCh,
		// so neither wait for it nor touch its VM — it runs its own pending
		// handlers as it unwinds.
		co.mu.Lock()
		started := co.started
		reapAbandoned := co.reapAbandoned
		co.mu.Unlock()
		if started && !reapAbandoned {
			<-co.doneCh
		}

		// Close any pending TBC vars on the coroutine VM.
		co.mu.Lock()
		coVM := co.coVM
		coErr := co.err
		co.err = nil
		co.mu.Unlock()
		if reapAbandoned {
			coVM = nil
		}
		if coVM != nil {
			var errVal vm.Value
			if coErr != nil {
				if le, ok := coErr.(*vm.LuaError); ok {
					errVal = le.Value
				} else {
					errVal = vm.NewString(coErr.Error())
				}
			}
			// coroutine.close on a non-errored coroutine performs a normal close
			// (no error object); only an actual pending error makes hasError true.
			if closeErr := coVM.ClosePendingTBC(errVal, coErr != nil); closeErr != nil {
				coErr = closeErr
			}
			// Clear the call stack so debug.getinfo(co, level) returns nil,
			// while keeping VMRef alive so gethook/isyieldable still work.
			coVM.ClearCallStack()
			// The handlers ran on this goroutine against the dead coroutine's
			// VM, so their work is charged there: publish it into the shared
			// budget and adopt it, or a script could buy unbounded __close time
			// by closing dead coroutine after dead coroutine.
			coVM.SyncInstructionBudget()
			v.SyncInstructionBudget()
		}
		if coErr != nil {
			v.Set(0, vm.False)
			var errVal vm.Value
			if le, ok := coErr.(*vm.LuaError); ok {
				errVal = le.Value
			} else {
				errVal = vm.NewString(coErr.Error())
			}
			// Lua 5.5: a nil error object is replaced by "<no error object>",
			// symmetrically with the resume path (and pcall/xpcall).
			if errVal.IsNil() {
				errVal = vm.NewString("<no error object>")
			}
			v.Set(1, errVal)
			return 2
		}
		v.Set(0, vm.True)
		return 1
	}

	// Track close-chain depth to detect "C stack overflow" when
	// __close handlers recursively call coroutine.close.
	if v.EnterCloseChain() {
		v.ExitCloseChain()
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("C stack overflow"))
		return 2
	}
	defer v.ExitCloseChain()

	// Mark as running during __close metamethod execution and flag
	// resumeCh as closed so that a concurrent coResume will not send.
	// Lua 5.4: coroutine.status reports "running" while __close handlers run.
	co.mu.Lock()
	co.status = statusRunning
	co.resumeChClosed = true
	co.mu.Unlock()

	// If the coroutine goroutine is blocked on resumeCh, unblock it
	// by closing the channel. coYield detects the closed channel and
	// calls CloseAllTBC() which runs __close handlers. If any handler
	// errors, CloseAllTBC re-raises the panic, which runCoroutine
	// catches and stores as co.err.
	// Safe: no concurrent send is possible because resumeChClosed is set.
	if co.started {
		close(co.resumeCh)
		<-co.doneCh
		// The __close handlers ran on the coroutine's goroutine; charge that
		// work to the family budget before we continue.
		v.SyncInstructionBudget()
	}

	// Now mark as dead and remove from registry
	co.mu.Lock()
	co.status = statusDead
	co.mu.Unlock()

	reg.mu.Lock()
	delete(reg.coroutines, int(id))
	reg.mu.Unlock()

	// Clear the call stack so debug.getinfo(co, level) returns nil,
	// while keeping VMRef alive so gethook/isyieldable still work.
	co.mu.Lock()
	coVM := co.coVM
	coErr := co.err
	co.mu.Unlock()
	if coVM != nil {
		coVM.ClearCallStack()
	}
	if coErr != nil {
		v.Set(0, vm.False)
		var errVal vm.Value
		if le, ok := coErr.(*vm.LuaError); ok {
			errVal = le.Value
		} else {
			errVal = vm.NewString(coErr.Error())
		}
		// Lua 5.5: a nil error object is replaced by "<no error object>",
		// symmetrically with the resume path (and pcall/xpcall).
		if errVal.IsNil() {
			errVal = vm.NewString("<no error object>")
		}
		v.Set(1, errVal)
		return 2
	}

	v.Set(0, vm.True)
	return 1
}

// coroutine.isyieldable([co]) -> boolean
func coIsYieldable(v *vm.VM) int {
	if v.ArgCount() >= 1 {
		arg := v.Get(1)
		if !arg.IsTable() || !arg.AsTable().IsThread() {
			callerArgError(v, 1, "coroutine.isyieldable", fmt.Sprintf("thread expected, got %s", arg.Type()))
		}
		tbl := arg.AsTable()
		// Check coroutine VM ref first (set after first resume)
		if coVM := tbl.VMRef(); coVM != nil {
			v.Set(0, vm.NewBool(coVM.IsYieldableContext()))
			return 1
		}
		// No VM ref yet — check coroutine status via ID. tbl is the LuaTable
		// interface here (no GetString), so use the cached key Value to avoid
		// boxing "__coroutine_id" into an interface on each call.
		idVal := tbl.Get(coIDKey)
		if !idVal.IsNil() {
			id, _ := idVal.ToInt()
			reg := getCoRegistry(v)
			reg.mu.Lock()
			co := reg.coroutines[int(id)]
			reg.mu.Unlock()
			if co != nil {
				co.mu.Lock()
				status := co.status
				co.mu.Unlock()
				// Suspended (never-resumed) coroutines are yieldable
				v.Set(0, vm.NewBool(status == statusSuspended))
				return 1
			}
			// Dead/collected coroutine
			v.Set(0, vm.True)
			return 1
		}
		// Thread table without coroutine ID — not yieldable
		v.Set(0, vm.False)
		return 1
	}
	v.Set(0, vm.NewBool(v.IsYieldableContext()))
	return 1
}
