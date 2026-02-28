package vm

import (
	"context"
	"fmt"

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
	tbcVars []int

	// Type metatables (for string, number, etc.)
	stringMeta LuaTable

	// Coroutine support
	yieldCh     chan []Value // Channel to send yield values (nil if not in coroutine)
	resumeCh    chan []Value // Channel to receive resume values (nil if not in coroutine)
	coroutineID int          // ID of the coroutine this VM belongs to (0 if main)
	threadObj   Value        // Thread object representing this VM (for coroutine.running)

	// Code loading support
	codeProvider LuaCodeProvider // Provider for loading Lua chunks (optional)
	vmID         string          // Optional identifier for this VM
	chunkName    string          // Name of the currently executing chunk

	// IO and OS provider support
	ioProvider LuaIoProvider // Provider for IO operations (optional)
	osProvider LuaOsProvider // Provider for OS operations (optional)

	// Debug provider support
	debugProvider LuaDebugProvider // Provider for diagnostic debug operations (optional)

	// Channel provider support
	chanProvider LuaChanProvider // Provider for channel operations (optional)

	// Time provider support
	timeProvider LuaTimeProvider // Provider for time operations (optional)

	// Print/warn provider support
	printProvider LuaPrintProvider // Provider for print/warn output routing (optional)
	warnEnabled   bool             // Per-VM warn flag (controlled by warn("@on")/"@off")

	// Execution control
	ctx        context.Context // nil = no cancellation checking
	limits     Limits          // zero values = no limit
	instrCount int64           // only tracked when MaxInstructions > 0

	// Output capture
	captureOutput bool      // When true, Print appends to outputLines instead of writing stdout
	outputLines   *[]string // Shared captured output buffer (pointer for coroutine sharing)
}

// Sentinel values for callFrame fields.
const (
	MultiReturn        = -1  // callFrame.nResults: return all results
	UseVMTop           = -1  // callFrame.argc: compute arg count from vm.top
	VarargBufferOffset = 256 // stack gap between MaxStack and vararg storage
)

// callFrame represents a function call on the call stack.
type callFrame struct {
	closure    *Closure // Function being executed
	pc         int      // Program counter (next instruction to execute)
	base       int      // Base stack index for this frame's registers
	nResults   int      // Expected number of results (MultiReturn = variable)
	isVararg   bool     // True if function is vararg
	varargPos  int      // Stack position where varargs start
	numVararg  int      // Number of varargs
	isTailCall bool     // True if this was a tail call
	argc       int      // Argument count for native functions (UseVMTop = use vm.top)
}

// New creates a new VM with an empty global environment.
// Optional VMOption arguments can configure context and limits.
func New(opts ...VMOption) *VM {
	vm := &VM{
		stack:       make([]Value, 256),
		globals:     NewEmptyTable(),
		warnEnabled: true,
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

// ProtectedCall calls a function in protected mode, catching any Lua errors.
// If fn is a table with a __call metamethod, the metamethod is invoked.
// Returns the function's results on success, or an error on failure.
// This is the Go equivalent of Lua's pcall().
func (vm *VM) ProtectedCall(fn Value, args []Value) (results []Value, err error) {
	// Save VM state for recovery
	savedTop := vm.top
	savedCallStackLen := len(vm.callStack)
	savedTbcLen := len(vm.tbcVars)
	savedOpenUpvaluesLen := len(vm.openUpvalues)

	defer func() {
		if r := recover(); r != nil {
			// Preserve LuaError so pcall/xpcall can return the original Lua value
			if le, ok := r.(*LuaError); ok {
				err = le
			} else {
				err = fmt.Errorf("%v", r)
			}
			results = nil
			// Restore call stack (but NOT vm.top yet — __close handlers need
			// the stack pointer past the TBC variables to avoid overwriting them)
			if len(vm.callStack) > savedCallStackLen {
				vm.callStack = vm.callStack[:savedCallStackLen]
			}
			// Close TBC variables that were created during the failed call.
			// Remove them from tbcVars BEFORE calling __close to prevent
			// double-close: the handler's OP_RETURN calls closeUpvalues which
			// would find the same entry still in tbcVars.
			tbcToClose := make([]int, len(vm.tbcVars)-savedTbcLen)
			copy(tbcToClose, vm.tbcVars[savedTbcLen:])
			vm.tbcVars = vm.tbcVars[:savedTbcLen]
			// Extract the Lua error value to pass to __close handlers
			var errVal Value
			if le, ok := r.(*LuaError); ok {
				errVal = le.Value
			} else {
				errVal = NewString(fmt.Sprintf("%v", r))
			}
			// Use callCloseHandlers to protect each __close call individually.
			// If a handler errors, it replaces err but other handlers still run.
			func() {
				defer func() {
					if closeR := recover(); closeR != nil {
						// __close error replaces original error
						if le, ok := closeR.(*LuaError); ok {
							err = le
						} else {
							err = fmt.Errorf("%v", closeR)
						}
					}
				}()
				vm.callCloseHandlers(tbcToClose, errVal)
			}()
			// Now restore vm.top after __close handlers are done
			vm.top = savedTop
			// Close upvalues
			for i := len(vm.openUpvalues) - 1; i >= savedOpenUpvaluesLen; i-- {
				vm.openUpvalues[i].Close()
			}
			vm.openUpvalues = vm.openUpvalues[:savedOpenUpvaluesLen]
		}
	}()

	if fn.IsNativeFunc() {
		// For native functions, we need to set up a temp frame
		nf := fn.AsNativeFunc()
		base := vm.top
		vm.ensureStack(base + len(args) + 10)

		// Copy arguments (slot 0 is reserved for the function, args start at 1)
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
			base: base,
			argc: len(args),
		})

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
		return vm.call(fn.AsClosure(), args, MultiReturn)
	}

	// Check for __call metamethod
	op := "__call"
	mm := vm.getMetafield(fn, op)
	if !mm.IsNil() {
		// New args: prepend fn (self)
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = fn
		copy(newArgs[1:], args)
		return vm.ProtectedCall(mm, newArgs)
	}

	return nil, vm.runtimeError("attempt to call a %s value", fn.Type())
}

// NewCoroutineVM creates a new VM for running a coroutine. The child VM shares
// the parent's globals, string metatable, providers, limits, and output buffer,
// but has its own stack, call stack, and upvalue tracking. Communication between
// parent and child happens through the yieldCh and resumeCh channels.
func NewCoroutineVM(parent *VM, yieldCh, resumeCh chan []Value, coID int) *VM {
	return &VM{
		stack:         make([]Value, 256),
		globals:       parent.globals,
		stringMeta:    parent.stringMeta,
		yieldCh:       yieldCh,
		resumeCh:      resumeCh,
		coroutineID:   coID,
		codeProvider:  parent.codeProvider,
		vmID:          parent.vmID,
		chunkName:     parent.chunkName,
		ioProvider:    parent.ioProvider,
		osProvider:    parent.osProvider,
		debugProvider: parent.debugProvider,
		chanProvider:  parent.chanProvider,
		timeProvider:  parent.timeProvider,
		printProvider: parent.printProvider,
		warnEnabled:   parent.warnEnabled,
		ctx:           parent.ctx,
		limits:        parent.limits,
		captureOutput: parent.captureOutput,
		outputLines:   parent.outputLines,
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

// CoroutineID returns the coroutine ID for this VM (0 if main).
func (vm *VM) CoroutineID() int {
	return vm.coroutineID
}

// CallCoroutine calls a closure as a coroutine, with yield support.
func (vm *VM) CallCoroutine(closure *Closure, args []Value) ([]Value, error) {
	return vm.call(closure, args, MultiReturn)
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




// callMetamethod calls a metamethod with 2 arguments and returns the first result
func (vm *VM) callMetamethod(fn, arg1, arg2 Value) (Value, error) {
	if fn.IsFunction() {
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

		nativeFrame := callFrame{base: nativeBase}
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

	return Nil, nil
}

// callMetamethod3 calls a metamethod with 3 arguments
func (vm *VM) callMetamethod3(fn, arg1, arg2, arg3 Value) (Value, error) {
	if fn.IsFunction() {
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

		nativeFrame := callFrame{base: nativeBase}
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

	return Nil, nil
}

// getMetafield retrieves a metafield from a value's metatable
func (vm *VM) getMetafield(v Value, key string) Value {
	if v.IsTable() {
		if mt := v.AsTable().Metatable(); mt != nil {
			return mt.Get(NewString(key))
		}
	}
	if v.IsString() && vm.stringMeta != nil {
		return vm.stringMeta.Get(NewString(key))
	}
	return Nil
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
				return fmt.Sprintf("%s:%d", proto.Source, proto.Lines[pc])
			}
			return proto.Source
		}
		idx--
	}
	return ""
}

