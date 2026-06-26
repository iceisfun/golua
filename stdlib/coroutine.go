package stdlib

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/iceisfun/golua/vm"
)

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
	// See reapCoroutines and wontfix/coroutine-goroutine-leak (master).
	v.OnClose(func(ctx context.Context) { reapCoroutines(v, ctx) })
}

// reapCoroutines force-terminates every still-suspended coroutine tracked by the
// VM's registry, so their backing goroutines exit. Registered as a VM close hook
// (runs on VM.Close). Go cannot reap a parked goroutine, but closing its resume
// channel unblocks it; the goroutine then runs its <close> handlers and returns
// — the same mechanism coroutine.close() uses. This bounds the
// coroutine-goroutine leak to the VM's lifetime.
func reapCoroutines(v *vm.VM, ctx context.Context) {
	reg := getCoRegistry(v)
	reg.mu.Lock()
	pending := make([]*Coroutine, 0, len(reg.coroutines))
	for _, co := range reg.coroutines {
		pending = append(pending, co)
	}
	reg.coroutines = make(map[int]*Coroutine)
	reg.mu.Unlock()

	for _, co := range pending {
		reapCoroutine(co, ctx)
	}
}

// reapCoroutine terminates one suspended coroutine's goroutine. It mirrors the
// suspended-close path of coClose (close resumeCh -> the goroutine unwinds via
// CloseAllTBC and returns), but: (1) only acts on started, suspended coroutines
// with an un-closed resume channel; (2) recovers per-coroutine so one handler's
// error doesn't abort the sweep; (3) bounds the wait by ctx so a blocking
// <close> handler can't hang VM.Close.
func reapCoroutine(co *Coroutine, ctx context.Context) {
	defer func() { _ = recover() }()

	co.mu.Lock()
	reap := co.started && co.status == statusSuspended && !co.resumeChClosed
	if reap {
		co.status = statusRunning
		co.resumeChClosed = true
	}
	co.mu.Unlock()
	if !reap {
		return
	}

	close(co.resumeCh)
	var done <-chan struct{}
	if ctx != nil {
		done = ctx.Done()
	}
	select {
	case <-co.doneCh:
	case <-done:
	}

	co.mu.Lock()
	co.status = statusDead
	co.mu.Unlock()
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
	co.resumeCh <- args
	co.mu.Unlock()

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
			v.Set(0, vm.False)
			// Preserve the original Lua error value if available
			if le, ok := err.(*vm.LuaError); ok {
				v.Set(1, le.Value)
			} else {
				v.Set(1, vm.NewString(err.Error()))
			}
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
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("execution interrupted: "+v.Context().Err().Error()))
		return 2
	}
}

// runCoroutine runs the coroutine function in a goroutine
func runCoroutine(co *Coroutine) {
	defer func() {
		if r := recover(); r != nil {
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
		co.resumeCh <- args
		co.mu.Unlock()

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
					if closeErr := coVM.ClosePendingTBC(errVal); closeErr != nil {
						err = closeErr
					}
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
			panic("execution interrupted: " + v.Context().Err().Error())
		}
	}, 1) // 1 upvalue: the wrapped coroutine thread
	wrapper.SetNativeFuncUpvalue(1, vm.NewTable(coTable))

	v.Set(0, wrapper)
	return 1
}

// coroutine.close(co) -> ok [, errmsg]
func coClose(v *vm.VM) int {
	coTable := getThreadTable(v, 1, "coroutine.close")
	idVal := coTable.GetString("__coroutine_id")
	if idVal.IsNil() {
		callerArgError(v, 1, "coroutine.close", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	id, _ := idVal.ToInt()

	// Cannot close the running (main) thread
	if int(id) == v.CoroutineID() {
		panic("cannot close a running coroutine")
	}

	// The main thread (id=0) is never in the coroutines map.
	// When called from a coroutine, the main thread is in "normal" status.
	if int(id) == 0 {
		panic("cannot close a normal coroutine")
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

		// Close any pending TBC vars on the coroutine VM.
		co.mu.Lock()
		coVM := co.coVM
		coErr := co.err
		co.err = nil
		co.mu.Unlock()
		if coVM != nil {
			var errVal vm.Value
			if coErr != nil {
				if le, ok := coErr.(*vm.LuaError); ok {
					errVal = le.Value
				} else {
					errVal = vm.NewString(coErr.Error())
				}
			}
			if closeErr := coVM.ClosePendingTBC(errVal); closeErr != nil {
				coErr = closeErr
			}
			// Clear the call stack so debug.getinfo(co, level) returns nil,
			// while keeping VMRef alive so gethook/isyieldable still work.
			coVM.ClearCallStack()
		}
		if coErr != nil {
			v.Set(0, vm.False)
			if le, ok := coErr.(*vm.LuaError); ok {
				v.Set(1, le.Value)
			} else {
				v.Set(1, vm.NewString(coErr.Error()))
			}
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
		if le, ok := coErr.(*vm.LuaError); ok {
			v.Set(1, le.Value)
		} else {
			v.Set(1, vm.NewString(coErr.Error()))
		}
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
