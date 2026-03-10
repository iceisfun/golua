package vm

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/iceisfun/golua/compiler"
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

	// Type metatables (for string, number, bool, nil, function, thread)
	stringMeta   LuaTable
	numberMeta   LuaTable
	boolMeta     LuaTable
	nilMeta      LuaTable
	functionMeta LuaTable
	threadMeta   LuaTable

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

	// Hook support
	hookFunc     Value // Hook callback function
	hookMask     byte  // Bitmask of active hook events
	hookCount    int   // Instruction count interval for count hooks
	hookCounter  int   // Current counter (counts down to 0)
	inHook       bool  // Re-entrancy guard
	lastHookLine int   // Last line reported by line hook (-1 = none)

	// Pending call name hint for debug.getinfo name inference.
	// Set before calling vm.call() and consumed by vm.call().
	pendingCallName     string
	pendingCallNameWhat string

	// Message handler for xpcall: called inside ProtectedCall's recovery
	// BEFORE the call stack is truncated, so debug.traceback can see
	// the full stack. Set by xpcall and cleared after use.
	MsgHandler         Value
	MsgHandlerResult   Value
	MsgHandlerUsed     bool
	lastErrorCallStack []callFrame // saved call stack from the last error

	// Execution control
	ctx           context.Context // nil = no cancellation checking
	limits        Limits          // zero values = no limit
	instrCount    int64           // only tracked when MaxInstructions > 0
	callDepthBase int             // inherited call depth from parent VM (for coroutines)
	closeDepth    *int32          // shared counter tracking nested coroutine.close depth (atomic)
	metaCallDepth int             // >0 when inside a metamethod call chain (for "C stack overflow" message)

	// GC rate limiting
	lastLuaGC   time.Time // Last time ProcessGcFinalizers invoked runtime.GC()
	gcCallCount int       // Number of times runtime.GC() was actually invoked (for testing)

	// Output capture
	captureOutput bool      // When true, Print appends to outputLines instead of writing stdout
	outputLines   *[]string // Shared captured output buffer (pointer for coroutine sharing)

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
	closure      *Closure // Function being executed
	funcValue    Value    // Function value for debug info / native frames
	pc           int      // Program counter (next instruction to execute)
	base         int      // Base stack index for this frame's registers
	nResults     int      // Expected number of results (MultiReturn = variable)
	isVararg     bool     // True if function is vararg
	varargPos    int      // Stack position where varargs start
	numVararg    int      // Number of varargs
	isTailCall   bool     // True if this was a tail call
	argc         int      // Argument count for native functions (UseVMTop = use vm.top)
	callName     string   // Override name for debug.getinfo (e.g., "close" for __close)
	callNameWhat string   // Override nameWhat (e.g., "metamethod")
	ftransfer    int      // First "transfer" index for debug hooks (1-based, 0 = unavailable)
	ntransfer    int      // Number of transfer values for debug hooks
}

// New creates a new VM with an empty global environment.
// Optional VMOption arguments can configure context and limits.
func New(opts ...VMOption) *VM {
	vm := &VM{
		stack:       make([]Value, 256),
		callStack:   make([]callFrame, 0, 32),
		globals:     NewEmptyTable(),
		warnEnabled: false,
		closeDepth:  new(int32),
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
			vm.MsgHandlerResult = hResults[0]
		} else {
			vm.MsgHandlerResult = Nil
		}
		succeeded = true
		break
	}

	if !succeeded {
		vm.MsgHandlerResult = NewString("error in error handling")
	}

	vm.limits.MaxCallDepth = savedMaxCallDepth
	vm.callStack = truncatedStack
	vm.MsgHandlerUsed = true
}

// ProtectedCall calls a function in protected mode, catching any Lua errors.
// If fn is a table with a __call metamethod, the metamethod is invoked.
// Returns the function's results on success, or an error on failure.
// This is the Go equivalent of Lua's pcall().
func (vm *VM) ProtectedCall(fn Value, args []Value) (results []Value, err error) {
	if !vm.MsgHandler.IsNil() {
		// Avoid stale stacks from previous non-xpcall failures interfering with
		// current xpcall message-handler dispatch.
		vm.lastErrorCallStack = nil
	}

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
			// Preserve LuaError so pcall/xpcall can return the original Lua value
			// locatedMsg stores the file:line-prefixed error for non-LuaError panics,
			// used later by the xpcall message handler.
			var locatedMsg string
			if le, ok := r.(*LuaError); ok {
				err = le
			} else {
				// For string panics from native functions, add the calling
				// Lua frame's source:line: prefix (like Lua 5.4's luaG_addinfo).
				msg := fmt.Sprintf("%v", r)
				locatedMsg = vm.AddCallerLocation(msg)
				err = fmt.Errorf("%s", locatedMsg)
			}
			results = nil

			// Save the full call stack snapshot before truncation so the
			// xpcall message handler can see the stack at the error point.
			var errorCallStack []callFrame
			if vm.MsgHandler.IsNil() {
				vm.lastErrorCallStack = make([]callFrame, len(vm.callStack))
				copy(vm.lastErrorCallStack, vm.callStack)
			}
			if !vm.MsgHandler.IsNil() && vm.lastErrorCallStack == nil {
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
				msgh = vm.MsgHandler
				vm.MsgHandler = Nil
				var errVal Value
				if le, ok := r.(*LuaError); ok {
					errVal = le.Value
				} else {
					errVal = NewString(locatedMsg)
				}
				vm.callMsgHandler(msgh, errVal, errorCallStack)
			} else if !vm.MsgHandler.IsNil() && vm.lastErrorCallStack != nil {
				// callCloseHandlers from a prior level saved a __close error stack
				msgh = vm.MsgHandler
				vm.MsgHandler = Nil
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
				if vm.MsgHandlerUsed {
					closeErrVal = vm.MsgHandlerResult
				} else if le, ok := r.(*LuaError); ok {
					closeErrVal = le.Value
				} else {
					closeErrVal = NewString(fmt.Sprintf("%v", r))
				}
				if !msgh.IsNil() {
					vm.MsgHandler = msgh
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
				vm.MsgHandler = Nil

				if closeErrorOccurred && !msgh.IsNil() && vm.MsgHandlerUsed {
					err = &LuaError{Value: vm.MsgHandlerResult}
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
			base:      base,
			argc:      len(args),
			funcValue: fn,
		})

		// Advance vm.top past the arguments so that any metamethod
		// calls from within the native function (e.g. __lt in math.max)
		// allocate their frames AFTER these args, not overlapping them.
		vm.top = base + 1 + len(args)

		// Call native function
		nResults := nf(vm)

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

	// Check for __call metamethod
	op := "__call"
	mm := vm.getMetafield(fn, op)
	if !mm.IsNil() {
		// New args: prepend fn (self)
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = fn
		copy(newArgs[1:], args)
		vm.pendingCallName = "call"
		vm.pendingCallNameWhat = "metamethod"
		return vm.ProtectedCall(mm, newArgs)
	}

	return nil, vm.runtimeError("attempt to call a %s value", vm.ObjTypeName(fn))
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

	// Check for __call metamethod
	mm := vm.getMetafield(fn, "__call")
	if !mm.IsNil() {
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = fn
		copy(newArgs[1:], args)
		vm.callUnprotected(mm, newArgs)
		return
	}

	panic(vm.runtimeError("attempt to call a %s value", vm.ObjTypeName(fn)))
}

// NewCoroutineVM creates a new VM for running a coroutine. The child VM shares
// the parent's globals, string metatable, providers, limits, and output buffer,
// but has its own stack, call stack, and upvalue tracking. Communication between
// parent and child happens through the yieldCh and resumeCh channels.
func NewCoroutineVM(parent *VM, yieldCh, resumeCh chan []Value, coID int) *VM {
	return &VM{
		stack:           make([]Value, 256),
		callStack:       make([]callFrame, 0, 16),
		globals:         parent.globals,
		stringMeta:      parent.stringMeta,
		numberMeta:      parent.numberMeta,
		boolMeta:        parent.boolMeta,
		nilMeta:         parent.nilMeta,
		functionMeta:    parent.functionMeta,
		threadMeta:      parent.threadMeta,
		yieldCh:         yieldCh,
		resumeCh:        resumeCh,
		coroutineID:     coID,
		codeProvider:    parent.codeProvider,
		vmID:            parent.vmID,
		chunkName:       parent.chunkName,
		ioProvider:      parent.ioProvider,
		osProvider:      parent.osProvider,
		execProvider:    parent.execProvider,
		exitHandler:     parent.exitHandler,
		debugProvider:   parent.debugProvider,
		chanProvider:    parent.chanProvider,
		timeProvider:    parent.timeProvider,
		processProvider: parent.processProvider,
		loadLibProvider: parent.loadLibProvider,
		printProvider:   parent.printProvider,
		warnEnabled:     parent.warnEnabled,
		ctx:             parent.ctx,
		limits:          parent.limits,
		callDepthBase:   parent.callDepthBase + len(parent.callStack),
		closeDepth:      parent.closeDepth,
		captureOutput:   parent.captureOutput,
		outputLines:     parent.outputLines,
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
	}
}

// CoroutineID returns the coroutine ID for this VM (0 if main).
func (vm *VM) CoroutineID() int {
	return vm.coroutineID
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

// ThreadObj returns the thread object representing this VM (for coroutine.running).
func (vm *VM) ThreadObj() Value {
	return vm.threadObj
}

// SetThreadObj sets the thread object representing this VM.
func (vm *VM) SetThreadObj(v Value) {
	vm.threadObj = v
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
		panic(&LuaError{Value: NewString(vm.runtimeError(msg).Error())})
	}
}

// hasCFrame reports whether the current overflow involves native (C) function
// frames in the recent call chain, used to decide between "stack overflow"
// and "C stack overflow".
func (vm *VM) hasCFrame() bool {
	if vm.metaCallDepth > 0 {
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
		vm.pendingCallNameWhat = "metamethod"
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
			base:         nativeBase,
			argc:         2,
			funcValue:    fn,
			callName:     name,
			callNameWhat: "metamethod",
		}
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
		vm.pendingCallNameWhat = "metamethod"
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
			base:         nativeBase,
			argc:         3,
			funcValue:    fn,
			callName:     name,
			callNameWhat: "metamethod",
		}
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
	for depth := 0; depth < vm.MaxMetaDepth(); depth++ {
		mm := vm.getMetafield(cur, "__call")
		if mm.IsNil() {
			return Nil, vm.runtimeError("attempt to call a %s value (metamethod '%s')", vm.ObjTypeName(fn), name)
		}
		// Prepend cur as self
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = cur
		copy(newArgs[1:], args)
		args = newArgs

		if mm.IsFunction() {
			vm.pendingCallName = name
			vm.pendingCallNameWhat = "metamethod"
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
				callNameWhat: "metamethod",
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
	return Nil, vm.runtimeError("'__call' chain too long; possible loop")
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
