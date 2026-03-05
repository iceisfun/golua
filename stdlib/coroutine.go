package stdlib

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/iceisfun/golua/vm"
)

// ctxDone returns the Done channel from the VM's context, or nil if no context is set.
// A nil channel blocks forever in select, which is the correct fallback.
func ctxDone(v *vm.VM) <-chan struct{} {
	if ctx := v.Context(); ctx != nil {
		return ctx.Done()
	}
	return nil
}

// coroutineStatus represents the lifecycle state of a coroutine.
type coroutineStatus string

const (
	statusSuspended coroutineStatus = "suspended"
	statusRunning   coroutineStatus = "running"
	statusDead      coroutineStatus = "dead"
	statusNormal    coroutineStatus = "normal"
)

// Coroutine represents a Lua coroutine
type Coroutine struct {
	id       int
	fn       vm.Value      // The function to run
	status   coroutineStatus // lifecycle state
	started  bool          // Whether the goroutine has been started
	vm       *vm.VM        // Reference to the VM
	thread   vm.Value      // Thread object (table) for coroutine.running
	resumeCh chan []vm.Value // Channel to send resume args
	yieldCh  chan []vm.Value // Channel to receive yield values
	doneCh   chan struct{}   // Channel to signal completion
	result   []vm.Value      // Final return values
	err      error           // Error if panicked
	mu       sync.Mutex
}

var (
	coroutineID   int
	coroutinesMu  sync.Mutex
	coroutines    = make(map[int]*Coroutine)
)

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
	v.SetThreadObj(vm.NewTable(mainThread))
}

// coroutine.create(f) -> thread
func coCreate(v *vm.VM) int {
	fn := v.Get(1)
	if !fn.IsFunction() && !fn.IsNativeFunc() {
		panic("bad argument #1 to 'create' (function expected)")
	}

	coroutinesMu.Lock()
	coroutineID++
	id := coroutineID
	coroutinesMu.Unlock()

	co := &Coroutine{
		id:       id,
		fn:       fn,
		status:   statusSuspended,
		vm:       v,
		resumeCh: make(chan []vm.Value, 1),
		yieldCh:  make(chan []vm.Value, 1),
		doneCh:   make(chan struct{}),
	}

	coroutinesMu.Lock()
	coroutines[id] = co
	coroutinesMu.Unlock()

	// Create a table to represent the coroutine (with metatable for type identification)
	coTable := vm.NewEmptyTable()
	coTable.SetString("__coroutine_id", vm.NewInt(int64(id)))
	coTable.SetThread(true)

	threadVal := vm.NewTable(coTable)
	co.thread = threadVal

	v.Set(0, threadVal)
	return 1
}

// coroutine.resume(co [, val1, ...]) -> ok, results...
func coResume(v *vm.VM) int {
	coVal := v.Get(1)
	if !coVal.IsTable() {
		panic(fmt.Sprintf("bad argument #1 to 'resume' (thread expected, got %s)", coVal.Type()))
	}

	coTable := coVal.AsTable()
	idVal := coTable.Get(vm.NewString("__coroutine_id"))
	if idVal.IsNil() {
		panic("bad argument #1 to 'resume' (thread expected)")
	}

	id, _ := idVal.ToInt()
	coroutinesMu.Lock()
	co := coroutines[int(id)]
	coroutinesMu.Unlock()

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
		coroutinesMu.Lock()
		if caller := coroutines[callerID]; caller != nil {
			caller.mu.Lock()
			caller.status = statusNormal
			caller.mu.Unlock()
		}
		coroutinesMu.Unlock()
	}

	// Start the goroutine if this is the first resume
	co.mu.Lock()
	if !co.started {
		co.started = true
		co.status = statusRunning
		go runCoroutine(co)
	} else {
		co.status = statusRunning
	}
	co.mu.Unlock()

	// Send args to the coroutine
	co.resumeCh <- args

	// restoreCallerStatus restores the caller's coroutine status to running
	restoreCallerStatus := func() {
		if callerID != 0 {
			coroutinesMu.Lock()
			if caller := coroutines[callerID]; caller != nil {
				caller.mu.Lock()
				caller.status = statusRunning
				caller.mu.Unlock()
			}
			coroutinesMu.Unlock()
		}
	}

	// Wait for yield or completion
	select {
	case results := <-co.yieldCh:
		restoreCallerStatus()
		needed := v.Base() + 1 + len(results)
		if !v.CheckStack(needed) {
			v.Set(0, vm.False)
			v.Set(1, vm.NewString("stack overflow"))
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
			v.Set(1, vm.NewString("stack overflow"))
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

	// Create a fresh VM for this coroutine that shares globals but has its own stack
	coVM := vm.NewCoroutineVM(co.vm, co.yieldCh, co.resumeCh, co.id)
	coVM.SetThreadObj(co.thread)

	// Store VM reference on thread table for debug.getlocal/setlocal coroutine support
	if co.thread.IsTable() {
		if tbl, ok := co.thread.AsTable().(*vm.Table); ok {
			tbl.SetVMRef(coVM)
		}
	}

	if co.fn.IsFunction() {
		results, err = coVM.CallCoroutine(co.fn.AsClosure(), args)
	} else if co.fn.IsNativeFunc() {
		// Use coroutine VM so yield works from functions called by the native func
		results, err = coVM.ProtectedCall(co.fn, args)
	}

	co.mu.Lock()
	co.result = results
	if err != nil {
		co.err = err
	}
	co.status = statusDead
	co.mu.Unlock()
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
		panic("attempt to yield from outside a coroutine")
	}
	if !v.IsYieldableContext() {
		panic("attempt to yield across a C-call boundary")
	}

	// Mark coroutine as suspended before yielding
	coID := v.CoroutineID()
	if coID != 0 {
		coroutinesMu.Lock()
		if co := coroutines[coID]; co != nil {
			co.mu.Lock()
			co.status = statusSuspended
			co.mu.Unlock()
		}
		coroutinesMu.Unlock()
	}

	yieldCh <- values

	// Wait for resume - mark as running when we get it
	args, ok := <-resumeCh
	if !ok {
		// Channel closed by coClose — close TBC vars, then terminate
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
		coroutinesMu.Lock()
		if co := coroutines[coID]; co != nil {
			co.mu.Lock()
			co.status = statusRunning
			co.mu.Unlock()
		}
		coroutinesMu.Unlock()
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
	coVal := v.Get(1)
	if !coVal.IsTable() {
		panic("bad argument #1 to 'status' (thread expected)")
	}

	coTable := coVal.AsTable()
	idVal := coTable.Get(vm.NewString("__coroutine_id"))
	if idVal.IsNil() {
		panic("bad argument #1 to 'status' (thread expected)")
	}

	id, _ := idVal.ToInt()
	coroutinesMu.Lock()
	co := coroutines[int(id)]
	coroutinesMu.Unlock()

	if co == nil {
		v.Set(0, vm.NewString(string(statusDead)))
		return 1
	}

	co.mu.Lock()
	status := co.status
	co.mu.Unlock()

	v.Set(0, vm.NewString(string(status)))
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
		panic("bad argument #1 to 'wrap' (function expected)")
	}

	// Create the coroutine
	coroutinesMu.Lock()
	coroutineID++
	id := coroutineID
	coroutinesMu.Unlock()

	// Create a thread table for this coroutine (for coroutine.running inside it)
	coTable := vm.NewEmptyTable()
	coTable.SetString("__coroutine_id", vm.NewInt(int64(id)))
	coTable.SetThread(true)

	co := &Coroutine{
		id:       id,
		fn:       fn,
		status:   statusSuspended,
		vm:       v,
		thread:   vm.NewTable(coTable),
		resumeCh: make(chan []vm.Value, 1),
		yieldCh:  make(chan []vm.Value, 1),
		doneCh:   make(chan struct{}),
	}

	coroutinesMu.Lock()
	coroutines[id] = co
	coroutinesMu.Unlock()

	// Return a wrapper function that resumes the coroutine on each call.
	// Unlike coroutine.resume, errors are raised directly (not returned as false, err).
	wrapper := vm.NewNativeFunc(func(v *vm.VM) int {
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
			coroutinesMu.Lock()
			if caller := coroutines[callerID]; caller != nil {
				caller.mu.Lock()
				caller.status = statusNormal
				caller.mu.Unlock()
			}
			coroutinesMu.Unlock()
		}
		restoreCallerStatus := func() {
			if callerID != 0 {
				coroutinesMu.Lock()
				if caller := coroutines[callerID]; caller != nil {
					caller.mu.Lock()
					caller.status = statusRunning
					caller.mu.Unlock()
				}
				coroutinesMu.Unlock()
			}
		}

		// Start the goroutine if this is the first call
		co.mu.Lock()
		if !co.started {
			co.started = true
			co.status = statusRunning
			go runCoroutine(co)
		} else {
			co.status = statusRunning
		}
		co.mu.Unlock()

		// Send args to the coroutine
		co.resumeCh <- args

		// Wait for yield or completion
		select {
		case results := <-co.yieldCh:
			restoreCallerStatus()
			needed := v.Base() + len(results)
			if !v.CheckStack(needed) {
				panic("stack overflow")
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
			co.mu.Unlock()

			if err != nil {
				// Preserve the original error (including *LuaError)
				panic(err)
			}

			needed := v.Base() + len(result)
			if !v.CheckStack(needed) {
				panic("stack overflow")
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
	})

	v.Set(0, wrapper)
	return 1
}

// coroutine.close(co) -> ok [, errmsg]
func coClose(v *vm.VM) int {
	coVal := v.Get(1)
	if coVal.IsNil() {
		panic("bad argument #1 to 'close' (value expected)")
	}
	if !coVal.IsTable() {
		panic("bad argument #1 to 'close' (thread expected)")
	}

	coTable := coVal.AsTable()
	idVal := coTable.Get(vm.NewString("__coroutine_id"))
	if idVal.IsNil() {
		panic("bad argument #1 to 'close' (thread expected)")
	}

	id, _ := idVal.ToInt()

	// Cannot close the running (main) thread
	if int(id) == v.CoroutineID() {
		panic("cannot close a running coroutine")
	}

	coroutinesMu.Lock()
	co := coroutines[int(id)]
	coroutinesMu.Unlock()

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
		// Already dead — check if it died with an error. The first close
		// after an error returns false + error (Lua 5.4 behavior), then
		// clears the error so subsequent closes return true.
		co.mu.Lock()
		coErr := co.err
		co.err = nil
		co.mu.Unlock()
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

	// Mark as dead and clean up
	co.mu.Lock()
	co.status = statusDead
	co.mu.Unlock()

	// Remove from global map
	coroutinesMu.Lock()
	delete(coroutines, int(id))
	coroutinesMu.Unlock()

	// If the coroutine goroutine is blocked on resumeCh, unblock it
	// by closing the channel. coYield detects the closed channel and
	// calls CloseAllTBC() which runs __close handlers. If any handler
	// errors, CloseAllTBC re-raises the panic, which runCoroutine
	// catches and stores as co.err.
	if co.started {
		close(co.resumeCh)
		<-co.doneCh
	}

	// Check if __close handlers produced an error (e.g. from a nested
	// coroutine.close chain exceeding the depth limit).
	co.mu.Lock()
	coErr := co.err
	co.mu.Unlock()
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

// coroutine.isyieldable() -> boolean
func coIsYieldable(v *vm.VM) int {
	v.Set(0, vm.NewBool(v.IsYieldableContext()))
	return 1
}
