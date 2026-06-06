package vm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
)

// VM is the Lua virtual machine state. Each VM instance has its own stack,
// call stack, open upvalues, and to-be-closed variable tracking. The global
// table is shared between a parent VM and its coroutine VMs.
//
// Create a VM with [New] and execute code with [VM.Run] or [VM.ProtectedCall].
// Configure behavior with [VMOption] functions: [WithContext], [WithLimits],
// [WithCaptureOutput].
//
// Provider interfaces control access to system resources. Without providers,
// the corresponding stdlib modules (io, os, debug, chan, time) are not
// registered when [stdlib.Open] is called.
type VM struct {
	stack     []Value     // Value stack
	top       int         // Top of stack (first free slot)
	callStack []callFrame // Call stack
	globals   LuaTable    // Global environment (_G)

	// Open upvalues linked list (sorted by stack index, descending)
	openUpvalues []*Upvalue

	// To-be-closed variables (stack index -> true)
	tbcVars        []int
	skipTBCCleanup bool // when true, ProtectedCall error recovery skips TBC cleanup

	// closingCoroutine is set when a coroutine is being force-closed via
	// coroutine.close. When true, ProtectedCall re-panics all errors so
	// that __close handler errors propagate through any pcall/xpcall
	// boundaries in the coroutine's call stack.
	closingCoroutine bool

	// Type metatables (for string, number, bool, nil, function, thread,
	// and lightuserdata values from debug.upvalueid).
	stringMeta        LuaTable
	numberMeta        LuaTable
	boolMeta          LuaTable
	nilMeta           LuaTable
	functionMeta      LuaTable
	threadMeta        LuaTable
	lightUserdataMeta LuaTable

	// Coroutine support
	yieldCh     chan []Value // Channel to send yield values (nil if not in coroutine)
	resumeCh    chan []Value // Channel to receive resume values (nil if not in coroutine)
	coroutineID int          // ID of the coroutine this VM belongs to (0 if main)
	threadObj   Value        // Thread object representing this VM (for coroutine.running)
	// > 0 means current execution is inside a non-yieldable native callback
	// context (Lua "C-call boundary").
	nonYieldableDepth int
	// Stack of call-stack base depths captured by user-visible protected calls
	// (pcall/xpcall). Used by native libraries that need Lua-compatible behavior
	// differences between direct protected callees and nested calls.
	userProtectedBases []int
	// > 0 while pcall/xpcall is directly invoking load() as its callee.
	directProtectedLoadDepth int

	// Code loading support
	codeProvider LuaCodeProvider // Provider for loading Lua chunks (optional)
	vmID         string          // Optional identifier for this VM
	chunkName    string          // Name of the currently executing chunk

	// IO and OS provider support
	ioProvider   LuaIoProvider   // Provider for IO operations (optional)
	osProvider   LuaOsProvider   // Provider for OS operations (optional)
	execProvider LuaExecProvider // Provider for command execution (optional)
	exitHandler  LuaExitHandler  // Handler for os.exit (optional)

	// Debug provider support
	debugProvider LuaDebugProvider // Provider for diagnostic debug operations (optional)
	registry      LuaTable         // Debug registry table (created on first access)

	// Channel provider support
	chanProvider LuaChanProvider // Provider for channel operations (optional)

	// Time provider support
	timeProvider LuaTimeProvider // Provider for time operations (optional)

	// package.loadlib provider support
	loadLibProvider LuaLoadLibProvider

	// Process provider support
	processProvider LuaProcessProvider // Provider for process spawning (optional)

	// Print/warn provider support
	printProvider LuaPrintProvider // Provider for print/warn output routing (optional)
	warnEnabled   bool             // Per-VM warn flag (controlled by warn("@on")/"@off")

	// All registered providers for lifecycle management (Close)
	registeredProviders []any

	// Hook support
	hookFunc     Value // Hook callback function
	hookMask     byte  // Bitmask of active hook events
	hookCount    int   // Instruction count interval for count hooks
	hookCounter  int   // Current counter (counts down to 0)
	inHook       bool  // Re-entrancy guard
	lastHookLine int   // Last line reported by line hook (-1 = none)
	lastHookPC   int   // Last pc checked by line hook (for backward jump detection)

	// Pending call name hint for debug.getinfo name inference.
	// Set before calling vm.call() and consumed by vm.call().
	pendingCallName              string
	pendingCallNameWhat          string
	pendingSuppressTracebackName bool
	pendingExtraArgs             int // Extra args count from __call chains, consumed by vm.call()

	// Message handler for xpcall: called inside ProtectedCall's recovery
	// BEFORE the call stack is truncated, so debug.traceback can see
	// the full stack. Set by xpcall and cleared after use.
	msgHandler         Value
	msgHandlerResult   Value
	msgHandlerUsed     bool
	lastErrorCallStack []callFrame // saved call stack from the last error

	// inMsgHandler tracks whether we're inside an xpcall message handler.
	// When >0 and a stack overflow occurs inside a nested pcall, the pcall
	// returns "error in error handling" instead of the stack overflow error.
	// This matches Lua 5.4's luaE_checkcstack behavior where overflow inside
	// error recovery triggers LUA_ERRERR.
	inMsgHandler int

	// Execution control
	ctx           context.Context // nil = no cancellation checking
	limits        Limits          // zero values = no limit
	instrCount    int64           // only tracked when MaxInstructions > 0
	callDepthBase int             // inherited call depth from parent VM (for coroutines)
	closeDepth    *int32          // shared counter tracking nested coroutine.close depth (atomic)
	metaCallDepth int             // >0 when inside a metamethod call chain (for "C stack overflow" message)

	// GC rate limiting and mode tracking
	lastLuaGC     time.Time // Last time ProcessGcFinalizers invoked runtime.GC()
	gcCallCount   int       // Number of times runtime.GC() was actually invoked (for testing)
	gcStepCounter int       // Instruction counter for GC stepping (counts up to GCStepInterval)
	gcMode        string    // Current GC mode: "generational" (default) or "incremental"
	gcRunning     bool      // Whether GC is "running" (tracked for collectgarbage("isrunning"))

	// Output capture
	captureOutput bool      // When true, Print appends to outputLines instead of writing stdout
	outputLines   *[]string // Shared captured output buffer (pointer for coroutine sharing)

	// GC finalization queue shared between root VM and its coroutines.
	gcQueue *gcQueue

	// Per-VM state bag for stdlib packages (coroutine registry, channel registry, etc.).
	// Shared between root VM and its coroutines via pointer/reference.
	internalState map[string]any
	internalMu    *sync.Mutex

	// Close hooks run when Close() is called. NOT shared with coroutine VMs.
	closeHooks []func(context.Context)

	// Pre-allocated return value buffer for OP_RETURN/OP_RETURN1.
	// Since the VM processes one instruction at a time (no concurrency),
	// this buffer can be reused across returns without allocation.
	retBuf [8]Value
}

// Sentinel values for callFrame fields.
const (
	MultiReturn        = -1  // callFrame.nResults: return all results
	UseVMTop           = -1  // callFrame.argc: compute arg count from vm.top
	VarargBufferOffset = 256 // stack gap between MaxStack and vararg storage
)

// callFrame represents a function call on the call stack.
type callFrame struct {
	closure               *Closure // Function being executed
	funcValue             Value    // Function value for debug info / native frames
	pc                    int      // Program counter (next instruction to execute)
	base                  int      // Base stack index for this frame's registers
	nResults              int      // Expected number of results (MultiReturn = variable)
	isVararg              bool     // True if function is vararg
	varargPos             int      // Stack position where varargs start (legacy, used by debug)
	numVararg             int      // Number of varargs
	varargs               []Value  // Vararg values stored off-stack to prevent cross-frame overlap
	isTailCall            bool     // True if this was a tail call
	argc                  int      // Argument count for native functions (UseVMTop = use vm.top)
	callName              string   // Override name for debug.getinfo (e.g., "close" for __close)
	callNameWhat          string   // Override nameWhat (e.g., nwMetamethod)
	suppressTracebackName bool
	ftransfer             int // First "transfer" index for debug hooks (1-based, 0 = unavailable)
	ntransfer             int // Number of transfer values for debug hooks
	extraArgs             int // Number of extra arguments added by __call metamethod chains
}

// New creates a new VM with an empty global environment.
// Optional VMOption arguments can configure context and limits.
func New(opts ...VMOption) *VM {
	vm := &VM{
		stack:         make([]Value, 256),
		callStack:     make([]callFrame, 0, 32),
		globals:       NewEmptyTable(),
		warnEnabled:   false,
		closeDepth:    new(int32),
		gcMode:        "generational",
		gcRunning:     true,
		ctx:           context.Background(),
		gcQueue:       &gcQueue{},
		internalState: make(map[string]any),
		internalMu:    &sync.Mutex{},
	}
	for _, opt := range opts {
		opt(vm)
	}
	return vm
}

// Globals returns the global table.
func (vm *VM) Globals() LuaTable {
	return vm.globals
}

// SetGlobal sets a global variable.
// String keys are always valid, so this never returns an error.
func (vm *VM) SetGlobal(name string, value Value) {
	_ = vm.globals.Set(NewString(name), value) // string key cannot fail
}

// GetGlobal gets a global variable.
func (vm *VM) GetGlobal(name string) Value {
	return vm.globals.Get(NewString(name))
}

// Run executes a compiled prototype and returns the results. The prototype's
// first upvalue is automatically bound to the VM's global table (_ENV).
// Errors from the Lua program are caught and returned as Go errors.
func (vm *VM) Run(proto *compiler.Proto) ([]Value, error) {
	// Create main closure
	closure := NewClosure(proto)
	// The main chunk has _ENV as its first upvalue
	if len(proto.Upvalues) > 0 {
		// Create a closed upvalue containing globals
		closure.Upvalues[0] = &Upvalue{
			closed: NewTable(vm.globals),
			isOpen: false,
		}
	}

	return vm.ProtectedCall(NewFunction(closure), nil)
}

// GetMsgHandler returns the current xpcall message handler.
func (vm *VM) GetMsgHandler() Value { return vm.msgHandler }

// SetMsgHandler sets the xpcall message handler.
func (vm *VM) SetMsgHandler(h Value) { vm.msgHandler = h }

// GetMsgHandlerResult returns the result from the last message handler call.
func (vm *VM) GetMsgHandlerResult() Value { return vm.msgHandlerResult }

// SetMsgHandlerResult sets the message handler result.
func (vm *VM) SetMsgHandlerResult(r Value) { vm.msgHandlerResult = r }

// IsMsgHandlerUsed returns whether the message handler was invoked.
func (vm *VM) IsMsgHandlerUsed() bool { return vm.msgHandlerUsed }

// SetMsgHandlerUsed sets whether the message handler was invoked.
func (vm *VM) SetMsgHandlerUsed(used bool) { vm.msgHandlerUsed = used }

// callMsgHandler calls the xpcall message handler with retry logic.
// In Lua 5.4, if the message handler itself errors, it is re-invoked with
// the new error value. This continues until the handler succeeds or a
// stack overflow / recursion limit is hit (producing "error in error handling").
// The errorCallStack parameter is the call stack snapshot at the error point;
// it is temporarily swapped in so debug.traceback works inside the handler.
func (vm *VM) callMsgHandler(msgh Value, errVal Value, errorCallStack []callFrame) {
	truncatedStack := vm.callStack

	// Save and increase max call depth for handler headroom
	savedMaxCallDepth := vm.limits.MaxCallDepth
	if vm.limits.MaxCallDepth >= 0 {
		needed := vm.callDepthBase + len(errorCallStack) + 200
		if needed > vm.limits.MaxCallDepth {
			vm.limits.MaxCallDepth = needed
		}
	}

	// Track that we're inside a message handler so nested pcall can
	// detect stack overflow as "error in error handling".
	vm.inMsgHandler++
	defer func() { vm.inMsgHandler-- }()

	const maxRetries = 200
	curErrVal := errVal
	succeeded := false

	for attempt := 0; attempt < maxRetries; attempt++ {
		var hResults []Value
		var hErr error
		var panicked bool

		func() {
			defer func() {
				if hr := recover(); hr != nil {
					panicked = true
					if le, ok := hr.(*LuaError); ok {
						curErrVal = le.Value
					} else {
						msg := fmt.Sprintf("%v", hr)
						msg = vm.AddCallerLocation(msg)
						curErrVal = NewString(msg)
					}
				}
			}()
			vm.callStack = errorCallStack
			exit := vm.EnterNonYieldable()
			hResults, hErr = vm.ProtectedCall(msgh, []Value{curErrVal})
			exit()
		}()

		if panicked {
			// Handler panicked; retry with the new error value
			continue
		}
		if hErr != nil {
			// Handler returned error via ProtectedCall; extract and retry
			if le, ok := hErr.(*LuaError); ok {
				curErrVal = le.Value
			} else {
				curErrVal = NewString(hErr.Error())
			}
			continue
		}
		// Handler succeeded
		if len(hResults) > 0 {
			vm.msgHandlerResult = hResults[0]
		} else {
			vm.msgHandlerResult = Nil
		}
		succeeded = true
		break
	}

	if !succeeded {
		vm.msgHandlerResult = NewString("error in error handling")
	}

	vm.limits.MaxCallDepth = savedMaxCallDepth
	vm.callStack = truncatedStack
	vm.msgHandlerUsed = true
}

// ProtectedCall calls a function in protected mode, catching any Lua errors.
// If fn is a table with a __call metamethod, the metamethod is invoked.
// Returns the function's results on success, or an error on failure.
// This is the Go equivalent of Lua's pcall().
func (vm *VM) ProtectedCall(fn Value, args []Value) (results []Value, err error) {
	// Save VM state for recovery
	savedTop := vm.top
	savedCallStackLen := len(vm.callStack)
	savedOpenUpvaluesLen := len(vm.openUpvalues)
	// Save and clear skipTBCCleanup: this call's recovery uses the saved
	// value, but nested ProtectedCalls (from pcall) see false so they
	// still close TBC vars normally.
	skipTBC := vm.skipTBCCleanup
	vm.skipTBCCleanup = false

	defer func() {
		if r := recover(); r != nil {
			// LuaExitError must propagate through all ProtectedCall boundaries
			if _, isExit := r.(*LuaExitError); isExit {
				panic(r)
			}
			// CoroutineSelfClose (Lua 5.5: coroutine.close on the running
			// coroutine) propagates through pcall/xpcall — the long-jump
			// terminates the entire coroutine, not just the protected call.
			if _, isSelfClose := r.(CoroutineSelfClose); isSelfClose {
				panic(r)
			}
			// When force-closing a coroutine, __close errors must propagate
			// through pcall/xpcall boundaries to reach coroutine.close.
			if vm.closingCoroutine {
				panic(r)
			}
			// Hook errors are uncatchable by pcall/xpcall (Lua 5.4 semantics).
			// Re-panic so they propagate to the outermost ProtectedCall (Run).
			if he, isHook := r.(*luaHookError); isHook {
				if vm.InUserProtected() {
					// Inside Lua pcall/xpcall: re-panic to propagate through
					panic(r)
				}
				// At top level (Run): unwrap and handle as a normal error
				r = he.original
			}
			// Preserve LuaError so pcall/xpcall can return the original Lua value
			// locatedMsg stores the file:line-prefixed error for non-LuaError panics,
			// used later by the xpcall message handler.
			var locatedMsg string
			if le, ok := r.(*LuaError); ok {
				err = le
			} else if e, ok := r.(error); ok {
				// A Go error value is a runtime error (raised via runtimeError)
				// that was already positioned at raise time from the frame
				// current then — a native function's runtime error is bare,
				// matching Lua's luaG_runerror on a C frame. Adding the
				// caller's location here would wrongly prefix e.g. a C
				// iterator's "attempt to index a nil value" with the
				// for-loop's source:line.
				locatedMsg = e.Error()
				err = e
			} else {
				// A string panic is a luaL_error-style error from a stdlib
				// arg check; add the calling Lua frame's source:line: prefix
				// (like Lua 5.4's luaL_where(L,1) / luaG_addinfo).
				msg := fmt.Sprintf("%v", r)
				locatedMsg = vm.AddCallerLocation(msg)
				err = fmt.Errorf("%s", locatedMsg)
			}
			results = nil

			// When inside an xpcall message handler, a stack overflow
			// caught by a nested pcall becomes "error in error handling"
			// (matching Lua 5.4's LUA_ERRERR from luaE_checkcstack).
			if vm.inMsgHandler > 0 {
				if isStackOverflow(r) {
					err = fmt.Errorf("error in error handling")
					// Skip message handler logic — just restore state
					if len(vm.callStack) > savedCallStackLen {
						vm.callStack = vm.callStack[:savedCallStackLen]
					}
					if !skipTBC {
						vm.top = savedTop
						for i := len(vm.openUpvalues) - 1; i >= savedOpenUpvaluesLen; i-- {
							vm.openUpvalues[i].Close()
						}
						vm.openUpvalues = vm.openUpvalues[:savedOpenUpvaluesLen]
					}
					return
				}
			}

			// Save the full call stack snapshot before truncation so the
			// xpcall message handler can see the stack at the error point.
			var errorCallStack []callFrame
			// Only save if no inner ProtectedCall already captured a richer
			// snapshot (e.g. __tostring → error re-raised by luaToString).
			if vm.msgHandler.IsNil() && vm.lastErrorCallStack == nil {
				vm.lastErrorCallStack = make([]callFrame, len(vm.callStack))
				copy(vm.lastErrorCallStack, vm.callStack)
			}
			if !vm.msgHandler.IsNil() && vm.lastErrorCallStack == nil {
				errorCallStack = make([]callFrame, len(vm.callStack))
				copy(errorCallStack, vm.callStack)
			}

			// Restore call stack (but NOT vm.top yet — __close handlers need
			// the stack pointer past the TBC variables to avoid overwriting them)
			if len(vm.callStack) > savedCallStackLen {
				vm.callStack = vm.callStack[:savedCallStackLen]
			}

			// In Lua 5.4, the xpcall message handler runs BEFORE TBC variables
			// are closed, so it can inspect pre-cleanup state. If a __close
			// handler then errors, the message handler is called AGAIN with
			// the __close error, and that result replaces the original.
			var msgh Value
			if errorCallStack != nil {
				// Call message handler with original error first
				msgh = vm.msgHandler
				vm.msgHandler = Nil
				var errVal Value
				if le, ok := r.(*LuaError); ok {
					errVal = le.Value
				} else {
					errVal = NewString(locatedMsg)
				}
				vm.callMsgHandler(msgh, errVal, errorCallStack)
			} else if !vm.msgHandler.IsNil() && vm.lastErrorCallStack != nil {
				// callCloseHandlers from a prior level saved a __close error stack
				msgh = vm.msgHandler
				vm.msgHandler = Nil
				var errVal Value
				if le, ok := r.(*LuaError); ok {
					errVal = le.Value
				} else {
					errVal = NewString(locatedMsg)
				}
				closeErrStack := vm.lastErrorCallStack
				vm.lastErrorCallStack = nil
				vm.callMsgHandler(msgh, errVal, closeErrStack)
			}

			// Close TBC variables AFTER the message handler has run.
			if !skipTBC {
				// Remove to-be-closed variables created in frames at or above
				// savedTop BEFORE calling __close to prevent double-close.
				//
				// Using stack index level is more robust than relying on append-only
				// tbcVars growth, because close operations during unwinding may
				// mutate tbcVars ordering/length.
				var tbcToClose []int
				kept := vm.tbcVars[:0]
				for _, idx := range vm.tbcVars {
					if idx >= savedTop {
						tbcToClose = append(tbcToClose, idx)
					} else {
						kept = append(kept, idx)
					}
				}
				vm.tbcVars = kept
				// Extract the Lua error value to pass to __close handlers
				var closeErrVal Value
				if vm.msgHandlerUsed {
					closeErrVal = vm.msgHandlerResult
				} else if le, ok := r.(*LuaError); ok {
					closeErrVal = le.Value
				} else {
					closeErrVal = NewString(fmt.Sprintf("%v", r))
				}
				if !msgh.IsNil() {
					vm.msgHandler = msgh
				}
				// Use callCloseHandlers to protect each __close call individually.
				// If a handler errors, it replaces err but other handlers still run.
				var closeErrorOccurred bool
				func() {
					defer func() {
						if closeR := recover(); closeR != nil {
							closeErrorOccurred = true
							// __close error replaces original error
							if le, ok := closeR.(*LuaError); ok {
								err = le
							} else {
								err = fmt.Errorf("%v", closeR)
							}
						}
					}()
					vm.callCloseHandlers(tbcToClose, closeErrVal, !msgh.IsNil())
				}()
				vm.msgHandler = Nil

				if closeErrorOccurred && !msgh.IsNil() && vm.msgHandlerUsed {
					err = &LuaError{Value: vm.msgHandlerResult}
				}
			}
			// Restore vm.top after __close handlers are done.
			// When skipTBC is set (coroutine top-level), keep the stack
			// intact so ClosePendingTBC can access TBC values later.
			if !skipTBC {
				vm.top = savedTop
			}
			// Close upvalues (but not when skipTBC, since TBC vars may reference them)
			if !skipTBC {
				for i := len(vm.openUpvalues) - 1; i >= savedOpenUpvaluesLen; i-- {
					vm.openUpvalues[i].Close()
				}
				vm.openUpvalues = vm.openUpvalues[:savedOpenUpvaluesLen]
			}

		}
	}()

	if fn.IsNativeFunc() {
		// For native functions, we need to set up a temp frame
		nf := fn.AsNativeFunc()
		base := vm.top
		vm.ensureStack(base + len(args) + 10)

		// Copy arguments (slot 0 is reserved for the function, args start at 1)
		vm.stack[base] = fn
		for i, arg := range args {
			vm.stack[base+1+i] = arg
		}

		// Clear any slots beyond the arguments to prevent stale data from affecting
		// optional argument checks (e.g., if !v.Get(5).IsNil())
		clearStart := base + 1 + len(args)
		clearEnd := clearStart + 4
		if clearEnd > len(vm.stack) {
			clearEnd = len(vm.stack)
		}
		for i := clearStart; i < clearEnd; i++ {
			vm.stack[i] = Nil
		}

		// Push a call frame so Get/Set/ArgCount work correctly
		// argc stored so ArgCount doesn't depend on vm.top
		vm.callStack = append(vm.callStack, callFrame{
			base:                  base,
			argc:                  len(args),
			funcValue:             fn,
			suppressTracebackName: vm.pendingSuppressTracebackName,
		})
		vm.pendingSuppressTracebackName = false

		// Advance vm.top past the arguments so that any metamethod
		// calls from within the native function (e.g. __lt in math.max)
		// allocate their frames AFTER these args, not overlapping them.
		vm.top = base + 1 + len(args)

		// Fire call hook after frame is pushed (matching doCall behavior)
		vm.fireCallHook()

		// Call native function
		nResults := nf(vm)

		// Fire return hook before popping the frame
		vm.fireReturnHook()

		// Pop the call frame
		vm.callStack = vm.callStack[:len(vm.callStack)-1]

		// Collect results
		results = make([]Value, nResults)
		for i := 0; i < nResults; i++ {
			results[i] = vm.stack[base+i]
		}

		vm.top = savedTop
		return results, nil
	}

	if fn.IsFunction() {
		results, err = vm.call(fn.AsClosure(), args, MultiReturn)
		return results, err
	}

	// Resolve __call metamethod chain (up to MaxCallChainDepth levels)
	cur := fn
	for depth := 1; ; depth++ {
		mm := vm.getMetafield(cur, MetaCall)
		if mm.IsNil() {
			return nil, vm.runtimeError("attempt to call a %s value", vm.ObjTypeName(fn))
		}
		if depth > MaxCallChainDepth {
			return nil, vm.runtimeError("'__call' chain too long")
		}
		// Prepend cur as self
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = cur
		copy(newArgs[1:], args)
		args = newArgs

		if mm.IsNativeFunc() || mm.IsFunction() {
			vm.pendingCallName = "call"
			vm.pendingCallNameWhat = nwMetamethod
			fn = mm
			break
		}
		// mm is itself not directly callable; continue resolving
		cur = mm
	}

	// Dispatch the resolved function
	if fn.IsNativeFunc() {
		nf := fn.AsNativeFunc()
		base := vm.top
		vm.ensureStack(base + len(args) + 10)

		vm.stack[base] = fn
		for i, arg := range args {
			vm.stack[base+1+i] = arg
		}

		clearStart := base + 1 + len(args)
		clearEnd := clearStart + 4
		if clearEnd > len(vm.stack) {
			clearEnd = len(vm.stack)
		}
		for i := clearStart; i < clearEnd; i++ {
			vm.stack[i] = Nil
		}

		vm.callStack = append(vm.callStack, callFrame{
			base:                  base,
			argc:                  len(args),
			funcValue:             fn,
			suppressTracebackName: vm.pendingSuppressTracebackName,
		})
		vm.pendingSuppressTracebackName = false
		vm.top = base + 1 + len(args)
		vm.fireCallHook()

		nResults := nf(vm)
		vm.fireReturnHook()
		vm.callStack = vm.callStack[:len(vm.callStack)-1]

		results = make([]Value, nResults)
		for i := 0; i < nResults; i++ {
			results[i] = vm.stack[base+i]
		}

		vm.top = savedTop
		return results, nil
	}

	// fn.IsFunction()
	results, err = vm.call(fn.AsClosure(), args, MultiReturn)
	return results, err
}

// callUnprotected calls a function without catching errors, so panics
// (Lua errors) propagate to the enclosing pcall/xpcall. Used by hook dispatch
// to match Lua 5.4 semantics where hook errors are not silently swallowed.
func (vm *VM) callUnprotected(fn Value, args []Value) {
	if fn.IsNativeFunc() {
		nf := fn.AsNativeFunc()
		savedTop := vm.top
		base := vm.top
		vm.ensureStack(base + len(args) + 10)
		vm.stack[base] = fn

		for i, arg := range args {
			vm.stack[base+1+i] = arg
		}
		clearStart := base + 1 + len(args)
		clearEnd := clearStart + 4
		if clearEnd > len(vm.stack) {
			clearEnd = len(vm.stack)
		}
		for i := clearStart; i < clearEnd; i++ {
			vm.stack[i] = Nil
		}

		vm.callStack = append(vm.callStack, callFrame{
			base:      base,
			argc:      len(args),
			funcValue: fn,
		})
		// Advance vm.top past args so metamethod frames don't overlap.
		vm.top = base + 1 + len(args)
		nf(vm)
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
		vm.top = savedTop
		return
	}

	if fn.IsFunction() {
		vm.call(fn.AsClosure(), args, 0) //nolint:errcheck
		return
	}

	// Resolve __call metamethod chain (up to MaxCallChainDepth levels)
	origFn := fn
	cur := fn
	for depth := 1; ; depth++ {
		mm := vm.getMetafield(cur, MetaCall)
		if mm.IsNil() {
			panic(vm.runtimeError("attempt to call a %s value", vm.ObjTypeName(origFn)))
		}
		if depth > MaxCallChainDepth {
			panic(vm.runtimeError("'__call' chain too long"))
		}
		// Prepend cur as self
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = cur
		copy(newArgs[1:], args)
		args = newArgs

		if mm.IsNativeFunc() || mm.IsFunction() {
			fn = mm
			break
		}
		cur = mm
	}

	// Dispatch the resolved function
	if fn.IsNativeFunc() {
		nf := fn.AsNativeFunc()
		savedTop := vm.top
		base := vm.top
		vm.ensureStack(base + len(args) + 10)
		vm.stack[base] = fn
		for i, arg := range args {
			vm.stack[base+1+i] = arg
		}
		clearStart := base + 1 + len(args)
		clearEnd := clearStart + 4
		if clearEnd > len(vm.stack) {
			clearEnd = len(vm.stack)
		}
		for i := clearStart; i < clearEnd; i++ {
			vm.stack[i] = Nil
		}
		vm.callStack = append(vm.callStack, callFrame{
			base:      base,
			argc:      len(args),
			funcValue: fn,
		})
		vm.top = base + 1 + len(args)
		nf(vm)
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
		vm.top = savedTop
		return
	}

	vm.call(fn.AsClosure(), args, 0) //nolint:errcheck
}

// NewCoroutineVM creates a new VM for running a coroutine. The child VM shares
// the parent's globals, string metatable, providers, limits, and output buffer,
// but has its own stack, call stack, and upvalue tracking. Communication between
// parent and child happens through the yieldCh and resumeCh channels.
func NewCoroutineVM(parent *VM, yieldCh, resumeCh chan []Value, coID int) *VM {
	return &VM{
		stack:             make([]Value, 256),
		callStack:         make([]callFrame, 0, 16),
		globals:           parent.globals,
		stringMeta:        parent.stringMeta,
		numberMeta:        parent.numberMeta,
		boolMeta:          parent.boolMeta,
		nilMeta:           parent.nilMeta,
		functionMeta:      parent.functionMeta,
		threadMeta:        parent.threadMeta,
		lightUserdataMeta: parent.lightUserdataMeta,
		yieldCh:           yieldCh,
		resumeCh:          resumeCh,
		coroutineID:       coID,
		codeProvider:      parent.codeProvider,
		vmID:              parent.vmID,
		chunkName:         parent.chunkName,
		ioProvider:        parent.ioProvider,
		osProvider:        parent.osProvider,
		execProvider:      parent.execProvider,
		exitHandler:       parent.exitHandler,
		debugProvider:     parent.debugProvider,
		chanProvider:      parent.chanProvider,
		timeProvider:      parent.timeProvider,
		processProvider:   parent.processProvider,
		loadLibProvider:   parent.loadLibProvider,
		printProvider:     parent.printProvider,
		warnEnabled:       parent.warnEnabled,
		ctx:               parent.ctx,
		limits:            parent.limits,
		callDepthBase:     parent.callDepthBase + len(parent.callStack),
		closeDepth:        parent.closeDepth,
		captureOutput:     parent.captureOutput,
		outputLines:       parent.outputLines,
		gcQueue:           parent.gcQueue,
		internalState:     parent.internalState,
		internalMu:        parent.internalMu,
	}
}

// SetStringMeta sets the metatable for all strings.
func (vm *VM) SetStringMeta(mt LuaTable) {
	vm.stringMeta = mt
}

// StringMeta returns the string metatable.
func (vm *VM) StringMeta() LuaTable {
	return vm.stringMeta
}

// SetNumberMeta sets the metatable for all numbers.
func (vm *VM) SetNumberMeta(mt LuaTable) {
	vm.numberMeta = mt
}

// NumberMeta returns the number metatable.
func (vm *VM) NumberMeta() LuaTable {
	return vm.numberMeta
}

// SetBoolMeta sets the metatable for all booleans.
func (vm *VM) SetBoolMeta(mt LuaTable) {
	vm.boolMeta = mt
}

// BoolMeta returns the boolean metatable.
func (vm *VM) BoolMeta() LuaTable {
	return vm.boolMeta
}

// SetNilMeta sets the metatable for nil values.
func (vm *VM) SetNilMeta(mt LuaTable) {
	vm.nilMeta = mt
}

// NilMeta returns the nil metatable.
func (vm *VM) NilMeta() LuaTable {
	return vm.nilMeta
}

// SetFunctionMeta sets the metatable for all functions.
func (vm *VM) SetFunctionMeta(mt LuaTable) {
	vm.functionMeta = mt
}

// FunctionMeta returns the function metatable.
func (vm *VM) FunctionMeta() LuaTable {
	return vm.functionMeta
}

// GetTypeMeta returns the type metatable for a value.
// Returns nil for tables (use table's own metatable) and types without metatables set.
func (vm *VM) GetTypeMeta(v Value) LuaTable {
	if v.IsString() {
		return vm.stringMeta
	}
	if v.IsNumber() {
		return vm.numberMeta
	}
	if v.IsBool() {
		return vm.boolMeta
	}
	if v.IsNil() {
		return vm.nilMeta
	}
	if v.IsFunction() || v.IsNativeFunc() {
		return vm.functionMeta
	}
	if v.IsLightUserdata() {
		return vm.lightUserdataMeta
	}
	if v.IsTable() {
		if tbl, ok := v.ptr.(*Table); ok && tbl.IsThread() {
			return vm.threadMeta
		}
	}
	return nil
}

// SetTypeMeta sets the type metatable for the given value's type.
func (vm *VM) SetTypeMeta(v Value, mt LuaTable) {
	if v.IsTable() {
		if tbl, ok := v.ptr.(*Table); ok && tbl.IsThread() {
			vm.threadMeta = mt
			return
		}
		v.AsTable().SetMetatable(mt)
		return
	}
	if v.IsString() {
		vm.stringMeta = mt
	} else if v.IsNumber() {
		vm.numberMeta = mt
	} else if v.IsBool() {
		vm.boolMeta = mt
	} else if v.IsNil() {
		vm.nilMeta = mt
	} else if v.IsFunction() || v.IsNativeFunc() {
		vm.functionMeta = mt
	} else if v.IsLightUserdata() {
		vm.lightUserdataMeta = mt
	}
}

// CoroutineID returns the coroutine ID for this VM (0 if main).
func (vm *VM) CoroutineID() int {
	return vm.coroutineID
}

// EnterClosingCoroutine sets the closingCoroutine flag so that ProtectedCall
// re-panics all errors, allowing __close errors to propagate through
// pcall/xpcall boundaries during coroutine.close.
func (vm *VM) EnterClosingCoroutine() {
	vm.closingCoroutine = true
}

// CallCoroutine calls a closure as a coroutine, with yield support.
// Uses ProtectedCall to ensure TBC variables are closed on error.
func (vm *VM) CallCoroutine(closure *Closure, args []Value) ([]Value, error) {
	return vm.ProtectedCallCoroutine(NewFunction(closure), args)
}

// ProtectedCallCoroutine is like ProtectedCall but skips TBC cleanup on error.
// Lua 5.4: TBC vars in a coroutine are NOT closed when it dies from
// an unhandled error (only closed on GC or coroutine.close).
func (vm *VM) ProtectedCallCoroutine(fn Value, args []Value) ([]Value, error) {
	vm.skipTBCCleanup = true
	return vm.ProtectedCall(fn, args)
}

// ClosePendingTBC closes all pending to-be-closed variables on this VM.
// Used by coroutine.close and wrap error recovery. Returns the final error
// value (which may be from a __close handler if one errors).
func (vm *VM) ClosePendingTBC(errVal Value) (finalErr error) {
	if len(vm.tbcVars) == 0 {
		return nil
	}
	// When a coroutine is already dead/errored and coroutine.close runs its
	// pending __close handlers, Lua 5.4 does not expose the stale suspended
	// frames through debug APIs during the close callbacks.
	savedCallStack := vm.callStack
	vm.callStack = vm.callStack[:0]
	defer func() {
		vm.callStack = savedCallStack
	}()
	tbcToClose := make([]int, len(vm.tbcVars))
	copy(tbcToClose, vm.tbcVars)
	vm.tbcVars = vm.tbcVars[:0]
	func() {
		defer func() {
			if r := recover(); r != nil {
				if le, ok := r.(*LuaError); ok {
					finalErr = le
				} else {
					finalErr = fmt.Errorf("%v", r)
				}
			}
		}()
		vm.callCloseHandlers(tbcToClose, errVal, false)
	}()
	return finalErr
}

// ClearCallStack empties the call stack so debug.getinfo level queries return
// nil, while keeping the VM itself alive for hook queries and isyieldable.
func (vm *VM) ClearCallStack() {
	vm.callStack = vm.callStack[:0]
	vm.lastErrorCallStack = nil
	vm.top = 0
}

// ThreadObj returns the thread object representing this VM (for coroutine.running).
func (vm *VM) ThreadObj() Value {
	return vm.threadObj
}

// SetThreadObj sets the thread object representing this VM.
func (vm *VM) SetThreadObj(v Value) {
	vm.threadObj = v
}

// isStackOverflow checks whether a recovered panic value is a stack overflow error.
func isStackOverflow(r interface{}) bool {
	if le, ok := r.(*LuaError); ok {
		if le.Value.IsString() {
			s := le.Value.AsString()
			return strings.Contains(s, "stack overflow")
		}
	}
	if s, ok := r.(string); ok {
		return strings.Contains(s, "stack overflow")
	}
	return false
}

// GetCoroutineChannels returns the yield and resume channels if this VM is a coroutine.
func (vm *VM) GetCoroutineChannels() (yieldCh, resumeCh chan []Value) {
	return vm.yieldCh, vm.resumeCh
}

// maxCallDepth returns the effective call depth limit.
// Positive: use that value. Zero: use DefaultMaxCallDepth. Negative: unlimited.
func (vm *VM) maxCallDepth() int {
	if vm.limits.MaxCallDepth > 0 {
		return vm.limits.MaxCallDepth
	}
	if vm.limits.MaxCallDepth < 0 {
		return 0 // unlimited
	}
	return DefaultMaxCallDepth
}

// checkCallDepth panics with "stack overflow" if the effective call depth
// (callDepthBase + current callStack length) exceeds the limit. The panic
// is a *LuaError so pcall/xpcall can catch it. When the overflow occurs
// inside a metamethod or native function call chain, the message is
// "C stack overflow" to match Lua 5.4's behavior.
func (vm *VM) checkCallDepth() {
	max := vm.maxCallDepth()
	if max > 0 && vm.callDepthBase+len(vm.callStack) > max {
		msg := "stack overflow"
		if vm.hasCFrame() {
			msg = "C stack overflow"
		}
		panic(&LuaError{Value: NewString(vm.runtimeError("%s", msg).Error())})
	}
}

// hasCFrame reports whether the current overflow involves native (C) function
// frames in the recent call chain, used to decide between "stack overflow"
// and "C stack overflow".
func (vm *VM) hasCFrame() bool {
	if vm.metaCallDepth > 0 {
		return true
	}
	// If callDepthBase > 0, this VM inherited call depth from a parent VM
	// through a native function (e.g., coroutine.resume), so there are
	// native frames in the overflow chain even if this VM's stack is pure Lua.
	if vm.callDepthBase > 0 {
		return true
	}
	n := len(vm.callStack)
	limit := 10
	if limit > n {
		limit = n
	}
	for i := n - 1; i >= n-limit; i-- {
		if vm.callStack[i].closure == nil {
			return true
		}
	}
	return false
}

// EnterCloseChain atomically increments the shared close-depth counter and
// returns true if the new depth exceeds the call-depth limit (i.e. "C stack
// overflow"). The caller must call ExitCloseChain when done regardless of
// the return value.
func (vm *VM) EnterCloseChain() bool {
	depth := atomic.AddInt32(vm.closeDepth, 1)
	max := vm.maxCallDepth()
	return max > 0 && int(depth) > max
}

// ExitCloseChain decrements the shared close-depth counter.
func (vm *VM) ExitCloseChain() {
	atomic.AddInt32(vm.closeDepth, -1)
}

// callMetamethod calls a metamethod with 2 arguments and returns the first result.
// name is the metamethod name without "__" prefix (e.g. "index", "add").
func (vm *VM) callMetamethod(name string, fn, arg1, arg2 Value) (Value, error) {
	vm.metaCallDepth++
	defer func() { vm.metaCallDepth-- }()

	if fn.IsFunction() {
		vm.pendingCallName = name
		vm.pendingCallNameWhat = nwMetamethod
		// Some __close contexts should retain getinfo() naming but suppress the
		// synthetic "in metamethod 'close'" label in traceback output.
		results, err := vm.call(fn.AsClosure(), []Value{arg1, arg2}, 1)
		if err != nil {
			return Nil, err
		}
		if len(results) > 0 {
			return results[0], nil
		}
		return Nil, nil
	}

	if fn.IsNativeFunc() {
		// Save state
		savedTop := vm.top
		frame := &vm.callStack[len(vm.callStack)-1]

		// Set up for native call at top of stack
		nativeBase := vm.top
		vm.ensureStack(nativeBase + 10)
		vm.stack[nativeBase+1] = arg1
		vm.stack[nativeBase+2] = arg2

		nativeFrame := callFrame{
			base:                  nativeBase,
			argc:                  2,
			funcValue:             fn,
			callName:              name,
			callNameWhat:          nwMetamethod,
			suppressTracebackName: vm.pendingSuppressTracebackName,
		}
		vm.pendingSuppressTracebackName = false
		vm.callStack = append(vm.callStack, nativeFrame)
		vm.top = nativeBase + 3

		nResults := fn.AsNativeFunc()(vm)
		var result Value
		if nResults > 0 {
			result = vm.stack[nativeBase]
		} else {
			result = Nil
		}

		// Restore state
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
		vm.top = savedTop
		_ = frame // silence unused warning

		return result, nil
	}

	// Not a direct function — try __call metamethod chain (callable tables, etc.)
	result, err := vm.callValue(name, fn, []Value{arg1, arg2})
	if err != nil {
		return Nil, err
	}
	return result, nil
}

// callMetamethod3 calls a metamethod with 3 arguments.
// name is the metamethod name without "__" prefix (e.g. "newindex").
func (vm *VM) callMetamethod3(name string, fn, arg1, arg2, arg3 Value) (Value, error) {
	vm.metaCallDepth++
	defer func() { vm.metaCallDepth-- }()

	if fn.IsFunction() {
		vm.pendingCallName = name
		vm.pendingCallNameWhat = nwMetamethod
		results, err := vm.call(fn.AsClosure(), []Value{arg1, arg2, arg3}, 0)
		if err != nil {
			return Nil, err
		}
		if len(results) > 0 {
			return results[0], nil
		}
		return Nil, nil
	}

	if fn.IsNativeFunc() {
		// Save state
		savedTop := vm.top

		// Set up for native call at top of stack
		nativeBase := vm.top
		vm.ensureStack(nativeBase + 10)
		vm.stack[nativeBase+1] = arg1
		vm.stack[nativeBase+2] = arg2
		vm.stack[nativeBase+3] = arg3

		nativeFrame := callFrame{
			base:                  nativeBase,
			argc:                  3,
			funcValue:             fn,
			callName:              name,
			callNameWhat:          nwMetamethod,
			suppressTracebackName: vm.pendingSuppressTracebackName,
		}
		vm.pendingSuppressTracebackName = false
		vm.callStack = append(vm.callStack, nativeFrame)
		vm.top = nativeBase + 4

		nResults := fn.AsNativeFunc()(vm)
		var result Value
		if nResults > 0 {
			result = vm.stack[nativeBase]
		} else {
			result = Nil
		}

		// Restore state
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
		vm.top = savedTop

		return result, nil
	}

	// Not a direct function — try __call metamethod chain (callable tables, etc.)
	result, err := vm.callValue(name, fn, []Value{arg1, arg2, arg3})
	if err != nil {
		return Nil, err
	}
	return result, nil
}

// callValue calls a value that may be a callable table (table with __call metamethod).
// It resolves the __call chain and invokes the final function. If the value is not
// callable at all, it returns an error with the metamethod name context.
// name is the metamethod name without "__" prefix (used for error messages and debug info).
func (vm *VM) callValue(name string, fn Value, args []Value) (Value, error) {
	// Resolve __call chain: each level prepends self
	cur := fn
	for depth := 1; ; depth++ {
		mm := vm.getMetafield(cur, MetaCall)
		if mm.IsNil() {
			return Nil, vm.runtimeError("attempt to call a %s value (metamethod '%s')", vm.ObjTypeName(fn), name)
		}
		if depth > MaxCallChainDepth {
			return Nil, vm.runtimeError("'__call' chain too long")
		}
		// Prepend cur as self
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = cur
		copy(newArgs[1:], args)
		args = newArgs

		if mm.IsFunction() {
			vm.pendingCallName = name
			vm.pendingCallNameWhat = nwMetamethod
			results, err := vm.call(mm.AsClosure(), args, 1)
			if err != nil {
				return Nil, err
			}
			if len(results) > 0 {
				return results[0], nil
			}
			return Nil, nil
		}

		if mm.IsNativeFunc() {
			savedTop := vm.top
			nativeBase := vm.top
			vm.ensureStack(nativeBase + len(args) + 10)
			for i, a := range args {
				vm.stack[nativeBase+1+i] = a
			}
			nativeFrame := callFrame{
				base:         nativeBase,
				argc:         len(args),
				funcValue:    mm,
				callName:     name,
				callNameWhat: nwMetamethod,
			}
			vm.callStack = append(vm.callStack, nativeFrame)
			vm.top = nativeBase + 1 + len(args)

			nResults := mm.AsNativeFunc()(vm)
			var result Value
			if nResults > 0 {
				result = vm.stack[nativeBase]
			} else {
				result = Nil
			}

			vm.callStack = vm.callStack[:len(vm.callStack)-1]
			vm.top = savedTop
			return result, nil
		}

		// __call is itself not directly callable — continue resolving
		cur = mm
	}
}

// GetMetafield retrieves a metafield from a value's metatable (exported for stdlib use).
func (vm *VM) GetMetafield(v Value, key string) Value {
	return vm.getMetafield(v, key)
}

// getMetafield retrieves a metafield from a value's metatable
func (vm *VM) getMetafield(v Value, key string) Value {
	var mt LuaTable
	if v.IsTable() {
		if tbl, ok := v.ptr.(*Table); ok && tbl.IsThread() {
			mt = vm.threadMeta
		} else {
			mt = v.AsTable().Metatable()
		}
	} else if ud := v.AsUserdata(); ud != nil {
		mt = ud.Metatable()
	} else {
		mt = vm.GetTypeMeta(v)
	}
	if mt == nil {
		return Nil
	}
	// Fast path: concrete *Table avoids NewString allocation
	if ct, ok := mt.(*Table); ok {
		return ct.GetString(key)
	}
	return mt.Get(NewString(key))
}

// GetSourceLocation returns "source:line" for the given call stack level.
// Level 1 = the caller of the current function, level 2 = its caller, etc.
// Returns "" if the level is out of range or the frame at that level is native.
func (vm *VM) GetSourceLocation(level int) string {
	// callStack: len-1 is the current native frame (error()), we start from len-2.
	// Count all frames (native and Lua) to match Lua 5.4 level semantics.
	// If the frame at the target level is native (no Lua source), return "".
	idx := len(vm.callStack) - 2 // skip error() itself
	count := 0
	for idx >= 0 {
		count++
		if count == level {
			frame := vm.callStack[idx]
			if frame.closure == nil {
				return "" // native frame has no source
			}
			proto := frame.closure.Proto
			pc := frame.pc - 1
			if pc < 0 {
				pc = 0
			}
			if pc < len(proto.Lines) {
				return fmt.Sprintf("%s:%d", shortSrc(proto.Source), proto.Lines[pc])
			}
			// No line info (e.g., stripped function) — don't add source prefix.
			return ""
		}
		idx--
	}
	return ""
}
