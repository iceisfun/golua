package stdlib

import (
	"fmt"
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

// Coroutine represents a Lua coroutine
type Coroutine struct {
	id       int
	fn       vm.Value      // The function to run
	status   string        // "suspended", "running", "dead", "normal"
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
		status:   "suspended",
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

	threadVal := vm.NewTable(coTable)
	co.thread = threadVal

	v.Set(0, threadVal)
	return 1
}

// coroutine.resume(co [, val1, ...]) -> ok, results...
func coResume(v *vm.VM) int {
	coVal := v.Get(1)
	if !coVal.IsTable() {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("bad argument #1 to 'resume' (thread expected)"))
		return 2
	}

	coTable := coVal.AsTable()
	idVal := coTable.Get(vm.NewString("__coroutine_id"))
	if idVal.IsNil() {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("bad argument #1 to 'resume' (thread expected)"))
		return 2
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

	if status == "dead" {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("cannot resume dead coroutine"))
		return 2
	}

	if status == "running" {
		v.Set(0, vm.False)
		v.Set(1, vm.NewString("cannot resume non-suspended coroutine"))
		return 2
	}

	// Collect arguments
	argc := v.ArgCount()
	args := make([]vm.Value, argc-1)
	for i := 2; i <= argc; i++ {
		args[i-2] = v.Get(i)
	}

	// Start the goroutine if this is the first resume
	co.mu.Lock()
	if !co.started {
		co.started = true
		co.status = "running"
		go runCoroutine(co)
	} else {
		co.status = "running"
	}
	co.mu.Unlock()

	// Send args to the coroutine
	co.resumeCh <- args

	// Wait for yield or completion
	select {
	case results := <-co.yieldCh:
		v.EnsureStack(v.Base() + 1 + len(results))
		v.Set(0, vm.True)
		for i, r := range results {
			v.Set(i+1, r)
		}
		return 1 + len(results)
	case <-co.doneCh:
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

		v.EnsureStack(v.Base() + 1 + len(result))
		v.Set(0, vm.True)
		for i, r := range result {
			v.Set(i+1, r)
		}
		return 1 + len(result)
	case <-ctxDone(v):
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
			co.status = "dead"
			co.mu.Unlock()
		}
		close(co.doneCh)
	}()

	co.mu.Lock()
	co.status = "running"
	co.mu.Unlock()

	// Wait for first resume args
	args := <-co.resumeCh

	// Call the function
	var results []vm.Value
	var err error

	// Create a fresh VM for this coroutine that shares globals but has its own stack
	coVM := vm.NewCoroutineVM(co.vm, co.yieldCh, co.resumeCh, co.id)
	coVM.SetThreadObj(co.thread)

	if co.fn.IsFunction() {
		results, err = coVM.CallCoroutine(co.fn.AsClosure(), args)
	} else if co.fn.IsNativeFunc() {
		// Native functions can't yield, just call them
		results, err = co.vm.ProtectedCall(co.fn, args)
	}

	co.mu.Lock()
	co.result = results
	if err != nil {
		co.err = err
	}
	co.status = "dead"
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

	// Mark coroutine as suspended before yielding
	coID := v.CoroutineID()
	if coID != 0 {
		coroutinesMu.Lock()
		if co := coroutines[coID]; co != nil {
			co.mu.Lock()
			co.status = "suspended"
			co.mu.Unlock()
		}
		coroutinesMu.Unlock()
	}

	yieldCh <- values

	// Wait for resume - mark as running when we get it
	args := <-resumeCh

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
			co.status = "running"
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
		v.Set(0, vm.NewString("dead"))
		return 1
	}

	co.mu.Lock()
	status := co.status
	co.mu.Unlock()

	v.Set(0, vm.NewString(status))
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

	co := &Coroutine{
		id:       id,
		fn:       fn,
		status:   "suspended",
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

		if status == "dead" {
			panic("cannot resume dead coroutine")
		}

		// Collect arguments
		argc := v.ArgCount()
		args := make([]vm.Value, argc)
		for i := 1; i <= argc; i++ {
			args[i-1] = v.Get(i)
		}

		// Start the goroutine if this is the first call
		co.mu.Lock()
		if !co.started {
			co.started = true
			co.status = "running"
			go runCoroutine(co)
		} else {
			co.status = "running"
		}
		co.mu.Unlock()

		// Send args to the coroutine
		co.resumeCh <- args

		// Wait for yield or completion
		select {
		case results := <-co.yieldCh:
			v.EnsureStack(v.Base() + len(results))
			for i, r := range results {
				v.Set(i, r)
			}
			return len(results)
		case <-co.doneCh:
			co.mu.Lock()
			err := co.err
			result := co.result
			co.mu.Unlock()

			if err != nil {
				// Preserve the original error (including *LuaError)
				panic(err)
			}

			v.EnsureStack(v.Base() + len(result))
			for i, r := range result {
				v.Set(i, r)
			}
			return len(result)
		case <-ctxDone(v):
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

	if status == "running" {
		panic("cannot close a running coroutine")
	}

	if status == "dead" {
		v.Set(0, vm.True)
		return 1
	}

	// Mark as dead and clean up
	co.mu.Lock()
	co.status = "dead"
	co.mu.Unlock()

	// Remove from global map
	coroutinesMu.Lock()
	delete(coroutines, int(id))
	coroutinesMu.Unlock()

	// If the coroutine goroutine is blocked on resumeCh, unblock it
	// by closing the channel. The goroutine will panic and exit via
	// the deferred recover in runCoroutine.
	if co.started {
		// Send a nil signal and let the goroutine drain
		// Close resumeCh so the goroutine unblocks and exits
		select {
		case co.resumeCh <- nil:
			// Goroutine will receive nil args and likely panic,
			// which is caught by the deferred recover in runCoroutine
		default:
		}
		// Wait for the goroutine to finish
		<-co.doneCh
	}

	v.Set(0, vm.True)
	return 1
}

// coroutine.isyieldable() -> boolean
func coIsYieldable(v *vm.VM) int {
	// Check if we're inside a coroutine
	yieldCh, _ := v.GetCoroutineChannels()
	v.Set(0, vm.NewBool(yieldCh != nil))
	return 1
}
