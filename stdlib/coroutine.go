package stdlib

import (
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
	fn       vm.Value        // The function to run
	status   coroutineStatus // lifecycle state
	started  bool            // Whether the goroutine has been started
	vm       *vm.VM          // Reference to the VM
	coVM     *vm.VM          // The coroutine's own VM (set after first resume)
	thread   vm.Value        // Thread object (table) for coroutine.running
	resumeCh chan []vm.Value // Channel to send resume args
	yieldCh  chan []vm.Value // Channel to receive yield values
	doneCh   chan struct{}   // Channel to signal completion
	result   []vm.Value      // Final return values
	err      error           // Error if panicked
	mu       sync.Mutex
}

var (
	coroutineID  int
	coroutinesMu sync.Mutex
	coroutines   = make(map[int]*Coroutine)
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
	mainThread.SetVMRef(v)
	v.SetThreadObj(vm.NewTable(mainThread))
}

// coroutine.create(f) -> thread
func coCreate(v *vm.VM) int {
	fn := v.Get(1)
	if !fn.IsFunction() && !fn.IsNativeFunc() {
		callerArgError(v, 1, "coroutine.create", fmt.Sprintf("function expected, got %s", coArgType(v, 1)))
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
	coVal := v.Get(1)
	if !coVal.IsTable() {
		callerArgError(v, 1, "coroutine.resume", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	coTable := coVal.AsTable()
	idVal := coTable.Get(vm.NewString("__coroutine_id"))
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
		callerArgError(v, 1, "coroutine.status", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	coTable := coVal.AsTable()
	idVal := coTable.Get(vm.NewString("__coroutine_id"))
	if idVal.IsNil() {
		callerArgError(v, 1, "coroutine.status", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	id, _ := idVal.ToInt()

	// The main thread (id=0) is not in the coroutines map.
	// Its status depends on whether the caller is the main thread itself.
	if int(id) == 0 {
		if v.CoroutineID() == 0 {
			v.Set(0, vm.NewString(string(statusRunning)))
		} else {
			// Main thread is suspended while a coroutine runs
			v.Set(0, vm.NewString(string(statusNormal)))
		}
		return 1
	}

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
		callerArgError(v, 1, "coroutine.wrap", fmt.Sprintf("function expected, got %s", coArgType(v, 1)))
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

	coroutinesMu.Lock()
	coroutines[id] = co
	coroutinesMu.Unlock()

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
				// Lua 5.4: coroutine.wrap adds caller location to string
				// errors via luaL_where(L,1) before re-raising. For string
				// errors, use a plain string panic so the VM's panic handler
				// adds the caller prefix via addCallerLocation. For non-string
				// errors (tables, etc.), preserve as *LuaError.
				if le, ok := err.(*vm.LuaError); ok {
					if le.Value.IsString() {
						panic(le.Value.AsString())
					}
					panic(le)
				}
				panic(err.Error())
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

	v.Set(0, wrapper)
	return 1
}

// coroutine.close(co) -> ok [, errmsg]
func coClose(v *vm.VM) int {
	coVal := v.Get(1)
	if coVal.IsNil() {
		callerArgError(v, 1, "coroutine.close", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}
	if !coVal.IsTable() {
		callerArgError(v, 1, "coroutine.close", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
	}

	coTable := coVal.AsTable()
	idVal := coTable.Get(vm.NewString("__coroutine_id"))
	if idVal.IsNil() {
		callerArgError(v, 1, "coroutine.close", fmt.Sprintf("thread expected, got %s", coArgType(v, 1)))
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

	// Mark as running during __close metamethod execution.
	// Lua 5.4: coroutine.status reports "running" while __close handlers run.
	co.mu.Lock()
	co.status = statusRunning
	co.mu.Unlock()

	// If the coroutine goroutine is blocked on resumeCh, unblock it
	// by closing the channel. coYield detects the closed channel and
	// calls CloseAllTBC() which runs __close handlers. If any handler
	// errors, CloseAllTBC re-raises the panic, which runCoroutine
	// catches and stores as co.err.
	if co.started {
		close(co.resumeCh)
		<-co.doneCh
	}

	// Now mark as dead and remove from global map
	co.mu.Lock()
	co.status = statusDead
	co.mu.Unlock()

	coroutinesMu.Lock()
	delete(coroutines, int(id))
	coroutinesMu.Unlock()

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
		// No VM ref yet — check coroutine status via ID
		idVal := tbl.Get(vm.NewString("__coroutine_id"))
		if !idVal.IsNil() {
			id, _ := idVal.ToInt()
			coroutinesMu.Lock()
			co := coroutines[int(id)]
			coroutinesMu.Unlock()
			if co != nil {
				co.mu.Lock()
				status := co.status
				co.mu.Unlock()
				// Suspended (never-resumed) coroutines are yieldable
				v.Set(0, vm.NewBool(status == statusSuspended))
				return 1
			}
			// Dead/collected coroutine
			v.Set(0, vm.False)
			return 1
		}
		// Thread table without coroutine ID — not yieldable
		v.Set(0, vm.False)
		return 1
	}
	v.Set(0, vm.NewBool(v.IsYieldableContext()))
	return 1
}
