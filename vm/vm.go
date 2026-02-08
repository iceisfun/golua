package vm

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/iceisfun/golua/compiler"
)

// VM is the Lua virtual machine state.
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

	// Code loading support
	codeProvider LuaCodeProvider // Provider for loading Lua chunks (optional)
	vmID         string          // Optional identifier for this VM
	chunkName    string          // Name of the currently executing chunk

	// IO and OS provider support
	ioProvider LuaIoProvider  // Provider for IO operations (optional)
	osProvider LuaOsProvider  // Provider for OS operations (optional)

	// Debug provider support
	debugProvider LuaDebugProvider // Provider for diagnostic debug operations (optional)

	// Execution control
	ctx        context.Context // nil = no cancellation checking
	limits     Limits          // zero values = no limit
	instrCount int64           // only tracked when MaxInstructions > 0
}

// callFrame represents a function call on the call stack.
type callFrame struct {
	closure   *Closure  // Function being executed
	pc        int       // Program counter (next instruction to execute)
	base      int       // Base stack index for this frame's registers
	nResults  int       // Expected number of results (-1 = variable)
	isVararg  bool      // True if function is vararg
	varargPos int       // Stack position where varargs start
	numVararg int       // Number of varargs
	isTailCall bool     // True if this was a tail call
}

// New creates a new VM with an empty global environment.
// Optional VMOption arguments can configure context and limits.
func New(opts ...VMOption) *VM {
	vm := &VM{
		stack:   make([]Value, 256),
		globals: NewEmptyTable(),
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
func (vm *VM) SetGlobal(name string, value Value) {
	vm.globals.Set(NewString(name), value)
}

// GetGlobal gets a global variable.
func (vm *VM) GetGlobal(name string) Value {
	return vm.globals.Get(NewString(name))
}

// Run executes a compiled prototype and returns the results.
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

	return vm.call(closure, nil, -1)
}

// ProtectedCall calls a function in protected mode, catching any errors.
func (vm *VM) ProtectedCall(fn Value, args []Value) (results []Value, err error) {
	// Save VM state for recovery
	savedTop := vm.top
	savedCallStackLen := len(vm.callStack)
	savedTbcLen := len(vm.tbcVars)
	savedOpenUpvaluesLen := len(vm.openUpvalues)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
			results = nil
			// Restore VM state
			vm.top = savedTop
			if len(vm.callStack) > savedCallStackLen {
				vm.callStack = vm.callStack[:savedCallStackLen]
			}
			// Close TBC variables that were created during the failed call
			for i := len(vm.tbcVars) - 1; i >= savedTbcLen; i-- {
				vm.callCloseMetamethod(vm.tbcVars[i])
			}
			vm.tbcVars = vm.tbcVars[:savedTbcLen]
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

		// Copy arguments
		for i, arg := range args {
			vm.stack[base+1+i] = arg
		}
		vm.top = base + 1 + len(args)

		// Call native function
		nResults := nf(vm)

		// Collect results
		results = make([]Value, nResults)
		for i := 0; i < nResults; i++ {
			results[i] = vm.stack[base+i]
		}

		vm.top = savedTop
		return results, nil
	}

	if fn.IsFunction() {
		return vm.call(fn.AsClosure(), args, -1)
	}

	return nil, fmt.Errorf("attempt to call a %s value", fn.Type())
}

// NewCoroutineVM creates a new VM for running a coroutine.
// It shares globals with the parent but has its own stack and coroutine channels.
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
		ctx:           parent.ctx,
		limits:        parent.limits,
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
	return vm.call(closure, args, -1)
}

// GetCoroutineChannels returns the yield and resume channels if this VM is a coroutine.
func (vm *VM) GetCoroutineChannels() (yieldCh, resumeCh chan []Value) {
	return vm.yieldCh, vm.resumeCh
}

// call invokes a closure with the given arguments and returns results.
func (vm *VM) call(closure *Closure, args []Value, nResults int) ([]Value, error) {
	if vm.limits.MaxCallDepth > 0 && len(vm.callStack) >= vm.limits.MaxCallDepth {
		return nil, fmt.Errorf("call stack overflow: depth %d exceeds limit %d",
			len(vm.callStack)+1, vm.limits.MaxCallDepth)
	}

	proto := closure.Proto

	// Set up the stack frame
	base := vm.top
	savedTop := vm.top
	vm.ensureStack(base + proto.MaxStack + 10)

	// Handle varargs
	numParams := proto.NumParams
	numArgs := len(args)
	varargPos := 0
	numVararg := 0

	if proto.IsVarArg {
		// Copy fixed params first
		for i := 0; i < numParams && i < numArgs; i++ {
			vm.stack[base+i] = args[i]
		}
		// Nil-fill missing fixed params
		for i := numArgs; i < numParams; i++ {
			vm.stack[base+i] = Nil
		}
		// Varargs go after fixed params conceptually, but we need to store them
		// In Lua 5.5, varargs are accessed via VARARG opcode
		if numArgs > numParams {
			// Store varargs well beyond the frame to prevent overlap with function return values
			// Native functions can return multiple values which get written to registers,
			// so we need a buffer between MaxStack and varargs
			varargPos = base + proto.MaxStack + 256
			numVararg = numArgs - numParams
			vm.ensureStack(varargPos + numVararg)
			for i := 0; i < numVararg; i++ {
				vm.stack[varargPos+i] = args[numParams+i]
			}
		}
	} else {
		// Non-vararg: copy args, nil-fill rest
		for i := 0; i < numParams && i < numArgs; i++ {
			vm.stack[base+i] = args[i]
		}
		for i := numArgs; i < numParams; i++ {
			vm.stack[base+i] = Nil
		}
	}

	// Push call frame
	frame := callFrame{
		closure:   closure,
		pc:        0,
		base:      base,
		nResults:  nResults,
		isVararg:  proto.IsVarArg,
		varargPos: varargPos,
		numVararg: numVararg,
	}
	vm.callStack = append(vm.callStack, frame)

	// Update vm.top to point past this frame's registers
	// This ensures nested calls get non-overlapping stack regions
	vm.top = base + proto.MaxStack

	// Execute
	results, err := vm.execute()

	// Pop call frame and restore top
	vm.callStack = vm.callStack[:len(vm.callStack)-1]
	vm.top = savedTop

	return results, err
}

// checkInterrupt checks for context cancellation and instruction limits.
// When neither ctx nor MaxInstructions is set, this is essentially free
// (two comparisons returning nil).
func (vm *VM) checkInterrupt() error {
	if vm.ctx != nil {
		if err := vm.ctx.Err(); err != nil {
			return fmt.Errorf("execution interrupted: %w", err)
		}
	}
	if vm.limits.MaxInstructions > 0 {
		vm.instrCount++
		if vm.instrCount > vm.limits.MaxInstructions {
			return fmt.Errorf("instruction limit exceeded: %d instructions",
				vm.limits.MaxInstructions)
		}
	}
	return nil
}

// execute runs the current call frame until it returns.
func (vm *VM) execute() ([]Value, error) {
	for {
		frame := &vm.callStack[len(vm.callStack)-1]
		proto := frame.closure.Proto
		code := proto.Code

		if frame.pc >= len(code) {
			return nil, nil
		}

		inst := code[frame.pc]
		frame.pc++

		op := inst.OpCode()

		switch op {
		case compiler.OP_MOVE:
			a, b := inst.A(), inst.B()
			vm.stack[frame.base+a] = vm.stack[frame.base+b]

		case compiler.OP_LOADI:
			a := inst.A()
			sbx := inst.SBx()
			vm.stack[frame.base+a] = NewInt(int64(sbx))

		case compiler.OP_LOADF:
			a := inst.A()
			sbx := inst.SBx()
			vm.stack[frame.base+a] = NewFloat(float64(sbx))

		case compiler.OP_LOADK:
			a, bx := inst.A(), inst.Bx()
			vm.stack[frame.base+a] = vm.constToValue(proto.Constants[bx])

		case compiler.OP_LOADKX:
			a := inst.A()
			// Next instruction is EXTRAARG with the constant index
			extra := code[frame.pc]
			frame.pc++
			ax := extra.Ax()
			vm.stack[frame.base+a] = vm.constToValue(proto.Constants[ax])

		case compiler.OP_LOADFALSE:
			a := inst.A()
			vm.stack[frame.base+a] = False

		case compiler.OP_LFALSESKIP:
			a := inst.A()
			vm.stack[frame.base+a] = False
			frame.pc++

		case compiler.OP_LOADTRUE:
			a := inst.A()
			vm.stack[frame.base+a] = True

		case compiler.OP_LOADNIL:
			a, b := inst.A(), inst.B()
			for i := a; i <= a+b; i++ {
				vm.stack[frame.base+i] = Nil
			}

		case compiler.OP_GETUPVAL:
			a, b := inst.A(), inst.B()
			vm.stack[frame.base+a] = frame.closure.Upvalues[b].Get()

		case compiler.OP_SETUPVAL:
			a, b := inst.A(), inst.B()
			frame.closure.Upvalues[b].Set(vm.stack[frame.base+a])

		case compiler.OP_GETTABUP:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := frame.closure.Upvalues[b].Get()
			key := vm.constToValue(proto.Constants[c])
			if t := table.AsTable(); t != nil {
				val, err := vm.tableGet(t, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_GETTABLE:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			key := vm.stack[frame.base+c]
			if t := table.AsTable(); t != nil {
				val, err := vm.tableGet(t, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
			} else if table.IsString() && vm.stringMeta != nil {
				// String indexing - use string metatable
				val := vm.stringMeta.Get(key)
				vm.stack[frame.base+a] = val
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_GETI:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			if t := table.AsTable(); t != nil {
				val, err := vm.tableGetInt(t, c)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_GETFIELD:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			key := proto.Constants[c].SVal
			if t := table.AsTable(); t != nil {
				val, err := vm.tableGetString(t, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
			} else if table.IsString() && vm.stringMeta != nil {
				// String field access - use string metatable
				val := vm.stringMeta.Get(NewString(key))
				vm.stack[frame.base+a] = val
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_SETTABUP:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := frame.closure.Upvalues[a].Get()
			key := vm.constToValue(proto.Constants[b])
			value := vm.getRK(frame, c, inst.K())
			if t := table.AsTable(); t != nil {
				if err := vm.tableSet(t, key, value); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_SETTABLE:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+a]
			key := vm.stack[frame.base+b]
			value := vm.getRK(frame, c, inst.K())
			if t := table.AsTable(); t != nil {
				if err := vm.tableSet(t, key, value); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_SETI:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+a]
			value := vm.getRK(frame, c, inst.K())
			if t := table.AsTable(); t != nil {
				if err := vm.tableSetInt(t, b, value); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_SETFIELD:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+a]
			key := proto.Constants[b].SVal
			value := vm.getRK(frame, c, inst.K())
			if t := table.AsTable(); t != nil {
				if err := vm.tableSetString(t, key, value); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_NEWTABLE:
			a := inst.A()
			// vB and vC encode size hints (we ignore them for now)
			vm.stack[frame.base+a] = NewTable(NewEmptyTable())

		case compiler.OP_SELF:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			vm.stack[frame.base+a+1] = table
			key := vm.constToValue(proto.Constants[c])
			if t := table.AsTable(); t != nil {
				vm.stack[frame.base+a] = t.Get(key)
			} else if table.IsString() && vm.stringMeta != nil {
				// String method call - use string metatable
				vm.stack[frame.base+a] = vm.stringMeta.Get(key)
			} else {
				return nil, fmt.Errorf("attempt to index a %s value", table.Type())
			}

		case compiler.OP_ADDI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			if n, ok := v.ToNumber(); ok {
				vm.stack[frame.base+a] = NewFloat(n + float64(sc))
			} else {
				return nil, fmt.Errorf("attempt to perform arithmetic on a %s value", v.Type())
			}

		case compiler.OP_ADDK, compiler.OP_SUBK, compiler.OP_MULK, compiler.OP_MODK,
			compiler.OP_POWK, compiler.OP_DIVK, compiler.OP_IDIVK:
			a, b, c := inst.A(), inst.B(), inst.C()
			v := vm.stack[frame.base+b]
			kv := vm.constToValue(proto.Constants[c])
			result, err := vm.arithK(op, v, kv)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_BANDK, compiler.OP_BORK, compiler.OP_BXORK:
			a, b, c := inst.A(), inst.B(), inst.C()
			v := vm.stack[frame.base+b]
			kv := vm.constToValue(proto.Constants[c])
			result, err := vm.bitwiseK(op, v, kv)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_SHLI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			if i, ok := v.ToInt(); ok {
				vm.stack[frame.base+a] = NewInt(int64(sc) << uint(i))
			} else {
				return nil, fmt.Errorf("attempt to perform bitwise operation on a %s value", v.Type())
			}

		case compiler.OP_SHRI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			if i, ok := v.ToInt(); ok {
				vm.stack[frame.base+a] = NewInt(i >> uint(sc))
			} else {
				return nil, fmt.Errorf("attempt to perform bitwise operation on a %s value", v.Type())
			}

		case compiler.OP_ADD, compiler.OP_SUB, compiler.OP_MUL, compiler.OP_MOD,
			compiler.OP_POW, compiler.OP_DIV, compiler.OP_IDIV:
			a, b, c := inst.A(), inst.B(), inst.C()
			v1 := vm.stack[frame.base+b]
			v2 := vm.stack[frame.base+c]
			result, err := vm.arith(op, v1, v2)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_BAND, compiler.OP_BOR, compiler.OP_BXOR,
			compiler.OP_SHL, compiler.OP_SHR:
			a, b, c := inst.A(), inst.B(), inst.C()
			v1 := vm.stack[frame.base+b]
			v2 := vm.stack[frame.base+c]
			result, err := vm.bitwise(op, v1, v2)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_MMBIN, compiler.OP_MMBINI, compiler.OP_MMBINK:
			// Metamethod calls - for now, skip (we'd need metatable support)
			// These are emitted after arithmetic ops to handle metamethods

		case compiler.OP_UNM:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]
			if v.IsNumber() {
				if v.IsInt() {
					vm.stack[frame.base+a] = NewInt(-v.AsInt())
				} else {
					vm.stack[frame.base+a] = NewFloat(-v.AsFloat())
				}
			} else {
				return nil, fmt.Errorf("attempt to perform arithmetic on a %s value", v.Type())
			}

		case compiler.OP_BNOT:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]
			if i, ok := v.ToInt(); ok {
				vm.stack[frame.base+a] = NewInt(^i)
			} else {
				return nil, fmt.Errorf("attempt to perform bitwise operation on a %s value", v.Type())
			}

		case compiler.OP_NOT:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]
			vm.stack[frame.base+a] = NewBool(!v.ToBool())

		case compiler.OP_LEN:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]
			switch {
			case v.IsString():
				vm.stack[frame.base+a] = NewInt(int64(len(v.AsString())))
			case v.IsTable():
				vm.stack[frame.base+a] = NewInt(int64(v.AsTable().Len()))
			default:
				return nil, fmt.Errorf("attempt to get length of a %s value", v.Type())
			}

		case compiler.OP_CONCAT:
			a, b := inst.A(), inst.B()
			// Concatenate b values starting at R[A]
			var builder strings.Builder
			for i := 0; i < b; i++ {
				v := vm.stack[frame.base+a+i]
				switch {
				case v.IsString():
					builder.WriteString(v.AsString())
				case v.IsNumber():
					builder.WriteString(v.String())
				default:
					return nil, fmt.Errorf("attempt to concatenate a %s value", v.Type())
				}
			}
			vm.stack[frame.base+a] = NewString(builder.String())

		case compiler.OP_CLOSE:
			a := inst.A()
			vm.closeUpvalues(frame.base + a)

		case compiler.OP_TBC:
			// Mark variable as to-be-closed
			a := inst.A()
			vm.tbcVars = append(vm.tbcVars, frame.base+a)

		case compiler.OP_JMP:
			sj := inst.SJ()
			if sj < 0 {
				if err := vm.checkInterrupt(); err != nil {
					return nil, err
				}
			}
			frame.pc += sj

		case compiler.OP_EQ:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := vm.stack[frame.base+b]
			eq := v1.Equal(v2)
			if eq != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LT:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := vm.stack[frame.base+b]
			lt, ok := v1.LessThan(v2)
			if !ok {
				return nil, fmt.Errorf("attempt to compare %s with %s", v1.Type(), v2.Type())
			}
			if lt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LE:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := vm.stack[frame.base+b]
			le, ok := v1.LessEqual(v2)
			if !ok {
				return nil, fmt.Errorf("attempt to compare %s with %s", v1.Type(), v2.Type())
			}
			if le != (k == 1) {
				frame.pc++
			}

		case compiler.OP_EQK:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := vm.constToValue(proto.Constants[b])
			eq := v1.Equal(v2)
			if eq != (k == 1) {
				frame.pc++
			}

		case compiler.OP_EQI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			eq := v.IsNumber() && v.AsFloat() == float64(sb)
			if eq != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LTI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			if !v.IsNumber() {
				return nil, fmt.Errorf("attempt to compare %s with number", v.Type())
			}
			lt := v.AsFloat() < float64(sb)
			if lt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LEI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			if !v.IsNumber() {
				return nil, fmt.Errorf("attempt to compare %s with number", v.Type())
			}
			le := v.AsFloat() <= float64(sb)
			if le != (k == 1) {
				frame.pc++
			}

		case compiler.OP_GTI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			if !v.IsNumber() {
				return nil, fmt.Errorf("attempt to compare %s with number", v.Type())
			}
			gt := v.AsFloat() > float64(sb)
			if gt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_GEI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			if !v.IsNumber() {
				return nil, fmt.Errorf("attempt to compare %s with number", v.Type())
			}
			ge := v.AsFloat() >= float64(sb)
			if ge != (k == 1) {
				frame.pc++
			}

		case compiler.OP_TEST:
			a, k := inst.A(), inst.K()
			v := vm.stack[frame.base+a]
			if v.ToBool() != (k == 1) {
				frame.pc++
			}

		case compiler.OP_TESTSET:
			a, b, k := inst.A(), inst.B(), inst.K()
			v := vm.stack[frame.base+b]
			if v.ToBool() != (k == 1) {
				frame.pc++
			} else {
				vm.stack[frame.base+a] = v
			}

		case compiler.OP_CALL:
			if err := vm.checkInterrupt(); err != nil {
				return nil, err
			}
			a, b, c := inst.A(), inst.B(), inst.C()
			results, err := vm.doCall(frame, a, b, c)
			if err != nil {
				return nil, err
			}
			_ = results

		case compiler.OP_TAILCALL:
			if err := vm.checkInterrupt(); err != nil {
				return nil, err
			}
			a, b, _ := inst.A(), inst.B(), inst.C()
			// Tail call optimization - reuse current frame
			fn := vm.stack[frame.base+a]

			// Collect arguments
			var args []Value
			if b == 0 {
				// Use all values from a+1 to top
				args = make([]Value, vm.top-(frame.base+a+1))
				copy(args, vm.stack[frame.base+a+1:vm.top])
			} else {
				args = make([]Value, b-1)
				copy(args, vm.stack[frame.base+a+1:frame.base+a+b])
			}

			// Close upvalues
			vm.closeUpvalues(frame.base)

			if fn.IsFunction() {
				closure := fn.AsClosure()
				// Reuse the frame
				frame.closure = closure
				frame.pc = 0
				proto := closure.Proto

				// Set up parameters
				numParams := proto.NumParams
				numArgs := len(args)

				if proto.IsVarArg {
					for i := 0; i < numParams && i < numArgs; i++ {
						vm.stack[frame.base+i] = args[i]
					}
					for i := numArgs; i < numParams; i++ {
						vm.stack[frame.base+i] = Nil
					}
					if numArgs > numParams {
						frame.varargPos = frame.base + proto.MaxStack
						frame.numVararg = numArgs - numParams
						vm.ensureStack(frame.varargPos + frame.numVararg)
						for i := 0; i < frame.numVararg; i++ {
							vm.stack[frame.varargPos+i] = args[numParams+i]
						}
					} else {
						frame.numVararg = 0
					}
					frame.isVararg = true
				} else {
					for i := 0; i < numParams && i < numArgs; i++ {
						vm.stack[frame.base+i] = args[i]
					}
					for i := numArgs; i < numParams; i++ {
						vm.stack[frame.base+i] = Nil
					}
				}
				continue
			} else if fn.IsNativeFunc() {
				// Native function tail call - can't truly optimize, just call
				nf := fn.AsNativeFunc()
				// Reuse current frame's base for the native call
				vm.stack[frame.base] = fn
				for i, arg := range args {
					vm.stack[frame.base+1+i] = arg
				}
				vm.top = frame.base + 1 + len(args)
				// The current frame already exists, just call the native function
				// vm.Base() will correctly return frame.base
				nResults := nf(vm)
				results := make([]Value, nResults)
				copy(results, vm.stack[frame.base:frame.base+nResults])
				return results, nil
			} else {
				return nil, fmt.Errorf("attempt to call a %s value", fn.Type())
			}

		case compiler.OP_RETURN:
			a, b, c := inst.A(), inst.B(), inst.C()
			_ = c // c contains info about closing upvalues

			// Close upvalues
			vm.closeUpvalues(frame.base)

			// Collect return values
			var results []Value
			if b == 0 {
				// Return values from a to top
				results = make([]Value, vm.top-(frame.base+a))
				copy(results, vm.stack[frame.base+a:vm.top])
			} else {
				results = make([]Value, b-1)
				copy(results, vm.stack[frame.base+a:frame.base+a+b-1])
			}
			return results, nil

		case compiler.OP_RETURN0:
			vm.closeUpvalues(frame.base)
			return nil, nil

		case compiler.OP_RETURN1:
			a := inst.A()
			vm.closeUpvalues(frame.base)
			return []Value{vm.stack[frame.base+a]}, nil

		case compiler.OP_FORLOOP:
			a, bx := inst.A(), inst.Bx()
			// R[A] = index, R[A+1] = limit, R[A+2] = step
			idx := vm.stack[frame.base+a].AsFloat()
			limit := vm.stack[frame.base+a+1].AsFloat()
			step := vm.stack[frame.base+a+2].AsFloat()

			idx += step
			vm.stack[frame.base+a] = NewFloat(idx)

			// Check if loop continues
			// Note: bx+1 accounts for pre-increment of frame.pc
			if step >= 0 {
				if idx <= limit {
					if err := vm.checkInterrupt(); err != nil {
						return nil, err
					}
					frame.pc -= bx + 1
					vm.stack[frame.base+a+3] = NewFloat(idx)
				}
			} else {
				if idx >= limit {
					if err := vm.checkInterrupt(); err != nil {
						return nil, err
					}
					frame.pc -= bx + 1
					vm.stack[frame.base+a+3] = NewFloat(idx)
				}
			}

		case compiler.OP_FORPREP:
			a, bx := inst.A(), inst.Bx()
			// R[A] = init, R[A+1] = limit, R[A+2] = step
			init := vm.stack[frame.base+a]
			limit := vm.stack[frame.base+a+1]
			step := vm.stack[frame.base+a+2]

			// Convert to numbers
			initF, ok1 := init.ToNumber()
			limitF, ok2 := limit.ToNumber()
			stepF, ok3 := step.ToNumber()
			if !ok1 || !ok2 || !ok3 {
				return nil, fmt.Errorf("'for' limit must be a number")
			}

			// Store as floats for consistency
			vm.stack[frame.base+a] = NewFloat(initF)
			vm.stack[frame.base+a+1] = NewFloat(limitF)
			vm.stack[frame.base+a+2] = NewFloat(stepF)

			// Check if loop should run at all
			if stepF >= 0 {
				if initF > limitF {
					frame.pc += bx + 1
				} else {
					vm.stack[frame.base+a+3] = NewFloat(initF)
				}
			} else {
				if initF < limitF {
					frame.pc += bx + 1
				} else {
					vm.stack[frame.base+a+3] = NewFloat(initF)
				}
			}

		case compiler.OP_TFORPREP:
			a, bx := inst.A(), inst.Bx()
			_ = a
			frame.pc += bx

		case compiler.OP_TFORCALL:
			a, c := inst.A(), inst.C()
			// R[A] = iterator function, R[A+1] = state, R[A+2] = control variable
			fn := vm.stack[frame.base+a]
			state := vm.stack[frame.base+a+1]
			ctrl := vm.stack[frame.base+a+2]

			// Call iterator: fn(state, ctrl)
			var results []Value
			var err error
			if fn.IsFunction() {
				results, err = vm.call(fn.AsClosure(), []Value{state, ctrl}, c)
			} else if fn.IsNativeFunc() {
				// Set up stack for native call at R[A+4] so results land there
				nativeBase := frame.base + a + 4
				vm.stack[nativeBase] = fn
				vm.stack[nativeBase+1] = state
				vm.stack[nativeBase+2] = ctrl

				// Push temporary call frame for native function
				nativeFrame := callFrame{
					base: nativeBase,
				}
				vm.callStack = append(vm.callStack, nativeFrame)

				oldTop := vm.top
				vm.top = nativeBase + 3

				nResults := fn.AsNativeFunc()(vm)
				results = make([]Value, nResults)
				copy(results, vm.stack[nativeBase:nativeBase+nResults])

				// Pop native frame and restore top
				vm.callStack = vm.callStack[:len(vm.callStack)-1]
				vm.top = oldTop
			} else {
				return nil, fmt.Errorf("attempt to call a %s value", fn.Type())
			}
			if err != nil {
				return nil, err
			}

			// Store results at R[A+4], ..., R[A+3+C]
			for i := 0; i < c && i < len(results); i++ {
				vm.stack[frame.base+a+4+i] = results[i]
			}
			for i := len(results); i < c; i++ {
				vm.stack[frame.base+a+4+i] = Nil
			}

		case compiler.OP_TFORLOOP:
			a, bx := inst.A(), inst.Bx()
			// If R[A+2] (first result, now at R[A+4] after TFORCALL) is not nil, continue
			// Note: bx+1 accounts for pre-increment of frame.pc
			if !vm.stack[frame.base+a+4].IsNil() {
				if err := vm.checkInterrupt(); err != nil {
					return nil, err
				}
				vm.stack[frame.base+a+2] = vm.stack[frame.base+a+4]
				frame.pc -= bx + 1
			}

		case compiler.OP_SETLIST:
			a := inst.A()
			// IvABC format: extract vB (count) and vC (offset)
			vB := int((uint32(inst) >> 16) & 0x3F)
			vC := int((uint32(inst) >> 22) & 0x3FF)
			k := inst.K()

			tbl := vm.stack[frame.base+a].AsTable()
			if tbl == nil {
				return nil, fmt.Errorf("attempt to index a non-table value")
			}

			n := vB
			if n == 0 {
				n = vm.top - (frame.base + a + 1)
			}

			offset := vC
			if k != 0 {
				// Extra arg contains offset
				extra := code[frame.pc]
				frame.pc++
				offset = extra.Ax()
			}

			// offset is the starting index (1-based for first batch)
			// Indices set are: offset, offset+1, ..., offset+n-1
			// Fast path: OP_SETLIST always follows OP_NEWTABLE, so table is always *Table
			if ct, ok := tbl.(*Table); ok {
				for i := 0; i < n; i++ {
					ct.SetInt(offset+i, vm.stack[frame.base+a+1+i])
				}
			} else {
				for i := 0; i < n; i++ {
					tbl.Set(NewInt(int64(offset+i)), vm.stack[frame.base+a+1+i])
				}
			}

		case compiler.OP_CLOSURE:
			a, bx := inst.A(), inst.Bx()
			subProto := proto.Protos[bx]
			newClosure := NewClosure(subProto)

			// Set up upvalues
			for i, uvDesc := range subProto.Upvalues {
				if uvDesc.InStack {
					// Capture from current stack
					idx := frame.base + uvDesc.Index
					newClosure.Upvalues[i] = vm.findOrCreateUpvalue(idx)
				} else {
					// Capture from enclosing closure's upvalues
					newClosure.Upvalues[i] = frame.closure.Upvalues[uvDesc.Index]
				}
			}

			vm.stack[frame.base+a] = NewFunction(newClosure)

		case compiler.OP_VARARG:
			a, c := inst.A(), inst.C()
			// Copy varargs to R[A], ..., R[A+C-2]
			// If C=0, copy all varargs and set top

			numWanted := c - 1
			if c == 0 {
				numWanted = frame.numVararg
				vm.top = frame.base + a + numWanted
			}

			for i := 0; i < numWanted; i++ {
				if i < frame.numVararg {
					vm.stack[frame.base+a+i] = vm.stack[frame.varargPos+i]
				} else {
					vm.stack[frame.base+a+i] = Nil
				}
			}

		case compiler.OP_VARARGPREP:
			// Handled during call setup

		case compiler.OP_EXTRAARG:
			// Used by other instructions, should not be executed directly

		default:
			return nil, fmt.Errorf("unimplemented opcode: %s", compiler.OpName(op))
		}
	}
}

// Helper methods

func (vm *VM) ensureStack(n int) {
	if vm.limits.MaxStackSlots > 0 && n >= vm.limits.MaxStackSlots {
		panic(fmt.Sprintf("stack overflow: %d slots exceeds limit %d",
			n, vm.limits.MaxStackSlots))
	}
	for len(vm.stack) <= n {
		vm.stack = append(vm.stack, make([]Value, 256)...)
	}
}

func (vm *VM) constToValue(c compiler.Value) Value {
	switch c.Type {
	case compiler.ValNil:
		return Nil
	case compiler.ValFalse:
		return False
	case compiler.ValTrue:
		return True
	case compiler.ValInt:
		return NewInt(c.IVal)
	case compiler.ValFloat:
		return NewFloat(c.FVal)
	case compiler.ValString:
		return NewString(c.SVal)
	default:
		return Nil
	}
}

func (vm *VM) getRK(frame *callFrame, c, k int) Value {
	proto := frame.closure.Proto
	if k != 0 {
		return vm.constToValue(proto.Constants[c])
	}
	return vm.stack[frame.base+c]
}

func (vm *VM) arith(op compiler.OpCode, v1, v2 Value) (Value, error) {
	n1, ok1 := v1.ToNumber()
	n2, ok2 := v2.ToNumber()

	// If both can be converted to numbers, do the arithmetic
	if ok1 && ok2 {
		var result float64
		switch op {
		case compiler.OP_ADD:
			result = n1 + n2
		case compiler.OP_SUB:
			result = n1 - n2
		case compiler.OP_MUL:
			result = n1 * n2
		case compiler.OP_DIV:
			result = n1 / n2
		case compiler.OP_IDIV:
			result = math.Floor(n1 / n2)
		case compiler.OP_MOD:
			result = math.Mod(n1, n2)
			// Lua mod: a % b = a - floor(a/b)*b
			if result != 0 && (result < 0) != (n2 < 0) {
				result += n2
			}
		case compiler.OP_POW:
			result = math.Pow(n1, n2)
		}

		// Try to return int if possible
		if v1.IsInt() && v2.IsInt() && op != compiler.OP_DIV && op != compiler.OP_POW {
			if i := int64(result); float64(i) == result {
				return NewInt(i), nil
			}
		}
		return NewFloat(result), nil
	}

	// Try metamethods
	mmName := vm.arithMetamethod(op)
	if mm := vm.getArithMetamethod(v1, v2, mmName); !mm.IsNil() {
		result, err := vm.callMetamethod(mm, v1, v2)
		if err != nil {
			return Nil, err
		}
		return result, nil
	}

	// No metamethod found, report error
	if !ok1 {
		return Nil, fmt.Errorf("attempt to perform arithmetic on a %s value", v1.Type())
	}
	return Nil, fmt.Errorf("attempt to perform arithmetic on a %s value", v2.Type())
}

// arithMetamethod returns the metamethod name for an arithmetic opcode
func (vm *VM) arithMetamethod(op compiler.OpCode) string {
	switch op {
	case compiler.OP_ADD, compiler.OP_ADDK:
		return "__add"
	case compiler.OP_SUB, compiler.OP_SUBK:
		return "__sub"
	case compiler.OP_MUL, compiler.OP_MULK:
		return "__mul"
	case compiler.OP_DIV, compiler.OP_DIVK:
		return "__div"
	case compiler.OP_IDIV, compiler.OP_IDIVK:
		return "__idiv"
	case compiler.OP_MOD, compiler.OP_MODK:
		return "__mod"
	case compiler.OP_POW, compiler.OP_POWK:
		return "__pow"
	default:
		return ""
	}
}

// getArithMetamethod looks for an arithmetic metamethod on either operand
func (vm *VM) getArithMetamethod(v1, v2 Value, name string) Value {
	nameVal := NewString(name)
	// Try first operand
	if v1.IsTable() {
		if mt := v1.AsTable().Metatable(); mt != nil {
			if mm := mt.Get(nameVal); !mm.IsNil() {
				return mm
			}
		}
	}
	// Try second operand
	if v2.IsTable() {
		if mt := v2.AsTable().Metatable(); mt != nil {
			if mm := mt.Get(nameVal); !mm.IsNil() {
				return mm
			}
		}
	}
	return Nil
}

func (vm *VM) arithK(op compiler.OpCode, v, kv Value) (Value, error) {
	n1, ok1 := v.ToNumber()
	n2, ok2 := kv.ToNumber()

	if ok1 && ok2 {
		var result float64
		switch op {
		case compiler.OP_ADDK:
			result = n1 + n2
		case compiler.OP_SUBK:
			result = n1 - n2
		case compiler.OP_MULK:
			result = n1 * n2
		case compiler.OP_DIVK:
			result = n1 / n2
		case compiler.OP_IDIVK:
			result = math.Floor(n1 / n2)
		case compiler.OP_MODK:
			result = math.Mod(n1, n2)
			if result != 0 && (result < 0) != (n2 < 0) {
				result += n2
			}
		case compiler.OP_POWK:
			result = math.Pow(n1, n2)
		}

		if v.IsInt() && kv.IsInt() && op != compiler.OP_DIVK && op != compiler.OP_POWK {
			if i := int64(result); float64(i) == result {
				return NewInt(i), nil
			}
		}
		return NewFloat(result), nil
	}

	// Try metamethods
	mmName := vm.arithMetamethod(op)
	if mm := vm.getArithMetamethod(v, kv, mmName); !mm.IsNil() {
		result, err := vm.callMetamethod(mm, v, kv)
		if err != nil {
			return Nil, err
		}
		return result, nil
	}

	if !ok1 {
		return Nil, fmt.Errorf("attempt to perform arithmetic on a %s value", v.Type())
	}
	return Nil, fmt.Errorf("attempt to perform arithmetic on a %s value", kv.Type())
}

func (vm *VM) bitwise(op compiler.OpCode, v1, v2 Value) (Value, error) {
	i1, ok1 := v1.ToInt()
	i2, ok2 := v2.ToInt()
	if !ok1 {
		return Nil, fmt.Errorf("attempt to perform bitwise operation on a %s value", v1.Type())
	}
	if !ok2 {
		return Nil, fmt.Errorf("attempt to perform bitwise operation on a %s value", v2.Type())
	}

	var result int64
	switch op {
	case compiler.OP_BAND:
		result = i1 & i2
	case compiler.OP_BOR:
		result = i1 | i2
	case compiler.OP_BXOR:
		result = i1 ^ i2
	case compiler.OP_SHL:
		if i2 >= 0 {
			result = i1 << uint(i2)
		} else {
			result = i1 >> uint(-i2)
		}
	case compiler.OP_SHR:
		if i2 >= 0 {
			result = i1 >> uint(i2)
		} else {
			result = i1 << uint(-i2)
		}
	}
	return NewInt(result), nil
}

func (vm *VM) bitwiseK(op compiler.OpCode, v, kv Value) (Value, error) {
	i1, ok1 := v.ToInt()
	i2, ok2 := kv.ToInt()
	if !ok1 {
		return Nil, fmt.Errorf("attempt to perform bitwise operation on a %s value", v.Type())
	}
	if !ok2 {
		return Nil, fmt.Errorf("attempt to perform bitwise operation on a %s value", kv.Type())
	}

	var result int64
	switch op {
	case compiler.OP_BANDK:
		result = i1 & i2
	case compiler.OP_BORK:
		result = i1 | i2
	case compiler.OP_BXORK:
		result = i1 ^ i2
	}
	return NewInt(result), nil
}

func (vm *VM) doCall(frame *callFrame, a, b, c int) ([]Value, error) {
	fn := vm.stack[frame.base+a]

	// Collect arguments
	var args []Value
	if b == 0 {
		// Use all values from a+1 to top
		args = make([]Value, vm.top-(frame.base+a+1))
		copy(args, vm.stack[frame.base+a+1:vm.top])
	} else {
		args = make([]Value, b-1)
		copy(args, vm.stack[frame.base+a+1:frame.base+a+b])
	}

	var results []Value
	var err error

	if fn.IsFunction() {
		// Set vm.top so the new frame starts after the function and its arguments
		// This prevents the nested call from overwriting caller's registers
		vm.top = frame.base + a + len(args) + 1
		results, err = vm.call(fn.AsClosure(), args, c-1)
	} else if fn.IsNativeFunc() {
		// Set up for native function
		// Push a temporary call frame so vm.Base() works correctly for native functions
		nativeBase := frame.base + a
		nativeFrame := callFrame{
			base: nativeBase,
		}
		vm.callStack = append(vm.callStack, nativeFrame)

		// Set vm.top for the native function (args start at base+1)
		oldTop := vm.top
		vm.top = nativeBase + 1 + len(args)

		// Clear any slots beyond the arguments to prevent stale data from affecting
		// optional argument checks (e.g., if !v.Get(3).IsNil())
		clearEnd := nativeBase + 1 + len(args) + 4 // Clear a few extra slots
		if clearEnd > len(vm.stack) {
			clearEnd = len(vm.stack)
		}
		for i := vm.top; i < clearEnd; i++ {
			vm.stack[i] = Nil
		}

		nResults := fn.AsNativeFunc()(vm)
		results = make([]Value, nResults)
		copy(results, vm.stack[nativeBase:nativeBase+nResults])

		// Pop the native frame and restore top
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
		vm.top = oldTop
	} else {
		return nil, fmt.Errorf("attempt to call a %s value", fn.Type())
	}

	if err != nil {
		return nil, err
	}

	// Store results
	nWanted := c - 1
	if c == 0 {
		// Variable results - set top
		nWanted = len(results)
		vm.top = frame.base + a + nWanted
	}

	for i := 0; i < nWanted; i++ {
		if i < len(results) {
			vm.stack[frame.base+a+i] = results[i]
		} else {
			vm.stack[frame.base+a+i] = Nil
		}
	}

	return results, nil
}

// Upvalue management

func (vm *VM) findOrCreateUpvalue(stackIdx int) *Upvalue {
	// Look for existing open upvalue at this index
	for _, uv := range vm.openUpvalues {
		if uv.stackIdx == stackIdx {
			return uv
		}
	}

	// Create new open upvalue
	uv := NewOpenUpvalue(vm, stackIdx)
	vm.openUpvalues = append(vm.openUpvalues, uv)
	return uv
}

func (vm *VM) closeUpvalues(level int) {
	// Close all upvalues with stack index >= level
	remaining := vm.openUpvalues[:0]
	for _, uv := range vm.openUpvalues {
		if uv.stackIdx >= level {
			uv.Close()
		} else {
			remaining = append(remaining, uv)
		}
	}
	vm.openUpvalues = remaining

	// Call __close metamethod on TBC variables in reverse order
	// (most recently declared first)
	remainingTBC := vm.tbcVars[:0]
	for i := len(vm.tbcVars) - 1; i >= 0; i-- {
		idx := vm.tbcVars[i]
		if idx >= level {
			vm.callCloseMetamethod(idx)
		} else {
			remainingTBC = append(remainingTBC, idx)
		}
	}
	// Reverse remainingTBC to restore original order
	for i, j := 0, len(remainingTBC)-1; i < j; i, j = i+1, j-1 {
		remainingTBC[i], remainingTBC[j] = remainingTBC[j], remainingTBC[i]
	}
	vm.tbcVars = remainingTBC
}

// callCloseMetamethod calls the __close metamethod on a TBC variable
func (vm *VM) callCloseMetamethod(stackIdx int) {
	val := vm.stack[stackIdx]
	if !val.IsTable() {
		return
	}
	t := val.AsTable()
	mt := t.Metatable()
	if mt == nil {
		return
	}
	closeFunc := mt.Get(metaClose)
	if closeFunc.IsNil() {
		return
	}
	// Call __close(val, nil) - second arg is error value (nil for normal exit)
	vm.callMetamethod(closeFunc, val, Nil)
}

// Stack access for native functions

// Base returns the base stack index for the current call.
func (vm *VM) Base() int {
	if len(vm.callStack) == 0 {
		return 0
	}
	return vm.callStack[len(vm.callStack)-1].base
}

// Top returns the top of the stack.
func (vm *VM) Top() int {
	return vm.top
}

// Get returns the value at the given stack index (relative to base).
func (vm *VM) Get(idx int) Value {
	base := vm.Base()
	if idx >= 0 {
		return vm.stack[base+idx]
	}
	// Negative index counts from top
	return vm.stack[vm.top+idx]
}

// Set sets the value at the given stack index (relative to base).
func (vm *VM) Set(idx int, v Value) {
	base := vm.Base()
	if idx >= 0 {
		vm.stack[base+idx] = v
	} else {
		vm.stack[vm.top+idx] = v
	}
}

// ArgCount returns the number of arguments passed to the current function.
func (vm *VM) ArgCount() int {
	if len(vm.callStack) == 0 {
		return 0
	}
	frame := &vm.callStack[len(vm.callStack)-1]
	return vm.top - frame.base - 1
}

// Push pushes a value onto the stack.
func (vm *VM) Push(v Value) {
	vm.ensureStack(vm.top + 1)
	vm.stack[vm.top] = v
	vm.top++
}

// Pop pops a value from the stack.
func (vm *VM) Pop() Value {
	vm.top--
	return vm.stack[vm.top]
}

// tableGet gets a value from a table, handling __index metamethod
func (vm *VM) tableGet(t LuaTable, key Value) (Value, error) {
	val := t.Get(key)
	if !val.IsNil() {
		return val, nil
	}

	// Key not found, check for __index metamethod
	mt := t.Metatable()
	if mt == nil {
		return Nil, nil
	}

	index := mt.Get(metaIndex)
	if index.IsNil() {
		return Nil, nil
	}

	if index.IsTable() {
		// __index is a table, recurse
		return vm.tableGet(index.AsTable(), key)
	}

	if index.IsFunction() || index.IsNativeFunc() {
		// __index is a function, call it with (table, key)
		return vm.callMetamethod(index, NewTable(t), key)
	}

	return Nil, nil
}

// tableGetString gets a value from a table by string key, handling __index metamethod
func (vm *VM) tableGetString(t LuaTable, key string) (Value, error) {
	val := t.Get(NewString(key))
	if !val.IsNil() {
		return val, nil
	}

	// Key not found, check for __index metamethod
	mt := t.Metatable()
	if mt == nil {
		return Nil, nil
	}

	index := mt.Get(metaIndex)
	if index.IsNil() {
		return Nil, nil
	}

	if index.IsTable() {
		// __index is a table, recurse
		return vm.tableGetString(index.AsTable(), key)
	}

	if index.IsFunction() || index.IsNativeFunc() {
		// __index is a function, call it with (table, key)
		return vm.callMetamethod(index, NewTable(t), NewString(key))
	}

	return Nil, nil
}

// tableGetInt gets a value from a table by int key, handling __index metamethod
func (vm *VM) tableGetInt(t LuaTable, key int) (Value, error) {
	val := t.Get(NewInt(int64(key)))
	if !val.IsNil() {
		return val, nil
	}

	// Key not found, check for __index metamethod
	mt := t.Metatable()
	if mt == nil {
		return Nil, nil
	}

	index := mt.Get(metaIndex)
	if index.IsNil() {
		return Nil, nil
	}

	if index.IsTable() {
		// __index is a table, recurse
		return vm.tableGetInt(index.AsTable(), key)
	}

	if index.IsFunction() || index.IsNativeFunc() {
		// __index is a function, call it with (table, key)
		return vm.callMetamethod(index, NewTable(t), NewInt(int64(key)))
	}

	return Nil, nil
}

// tableSet sets a value in a table, handling __newindex metamethod
func (vm *VM) tableSet(t LuaTable, key, value Value) error {
	// Check if key already exists (raw access)
	existing := t.Get(key)
	if !existing.IsNil() {
		// Key exists, set directly
		t.Set(key, value)
		return nil
	}

	// Key doesn't exist, check for __newindex metamethod
	mt := t.Metatable()
	if mt == nil {
		t.Set(key, value)
		return nil
	}

	newindex := mt.Get(metaNewIndex)
	if newindex.IsNil() {
		t.Set(key, value)
		return nil
	}

	if newindex.IsTable() {
		// __newindex is a table, set in that table instead
		return vm.tableSet(newindex.AsTable(), key, value)
	}

	if newindex.IsFunction() || newindex.IsNativeFunc() {
		// __newindex is a function, call it with (table, key, value)
		_, err := vm.callMetamethod3(newindex, NewTable(t), key, value)
		return err
	}

	t.Set(key, value)
	return nil
}

// tableSetString sets a value in a table by string key, handling __newindex metamethod
func (vm *VM) tableSetString(t LuaTable, key string, value Value) error {
	keyVal := NewString(key)
	// Check if key already exists (raw access)
	existing := t.Get(keyVal)
	if !existing.IsNil() {
		// Key exists, set directly
		t.Set(keyVal, value)
		return nil
	}

	// Key doesn't exist, check for __newindex metamethod
	mt := t.Metatable()
	if mt == nil {
		t.Set(keyVal, value)
		return nil
	}

	newindex := mt.Get(metaNewIndex)
	if newindex.IsNil() {
		t.Set(keyVal, value)
		return nil
	}

	if newindex.IsTable() {
		// __newindex is a table, set in that table instead
		return vm.tableSetString(newindex.AsTable(), key, value)
	}

	if newindex.IsFunction() || newindex.IsNativeFunc() {
		// __newindex is a function, call it with (table, key, value)
		_, err := vm.callMetamethod3(newindex, NewTable(t), keyVal, value)
		return err
	}

	t.Set(keyVal, value)
	return nil
}

// tableSetInt sets a value in a table by int key, handling __newindex metamethod
func (vm *VM) tableSetInt(t LuaTable, key int, value Value) error {
	keyVal := NewInt(int64(key))
	// Check if key already exists (raw access)
	existing := t.Get(keyVal)
	if !existing.IsNil() {
		// Key exists, set directly
		t.Set(keyVal, value)
		return nil
	}

	// Key doesn't exist, check for __newindex metamethod
	mt := t.Metatable()
	if mt == nil {
		t.Set(keyVal, value)
		return nil
	}

	newindex := mt.Get(metaNewIndex)
	if newindex.IsNil() {
		t.Set(keyVal, value)
		return nil
	}

	if newindex.IsTable() {
		// __newindex is a table, set in that table instead
		return vm.tableSetInt(newindex.AsTable(), key, value)
	}

	if newindex.IsFunction() || newindex.IsNativeFunc() {
		// __newindex is a function, call it with (table, key, value)
		_, err := vm.callMetamethod3(newindex, NewTable(t), keyVal, value)
		return err
	}

	t.Set(keyVal, value)
	return nil
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

// SetCodeProvider sets the code provider for this VM.
// The code provider is used by load, loadfile, and dofile to resolve and load Lua chunks.
func (vm *VM) SetCodeProvider(provider LuaCodeProvider) {
	vm.codeProvider = provider
}

// CodeProvider returns the current code provider, or nil if none is set.
func (vm *VM) CodeProvider() LuaCodeProvider {
	return vm.codeProvider
}

// SetVMID sets the VM identifier (used in caller context for code loading).
func (vm *VM) SetVMID(id string) {
	vm.vmID = id
}

// VMID returns the VM identifier.
func (vm *VM) VMID() string {
	return vm.vmID
}

// SetChunkName sets the name of the currently executing chunk.
func (vm *VM) SetChunkName(name string) {
	vm.chunkName = name
}

// ChunkName returns the name of the currently executing chunk.
func (vm *VM) ChunkName() string {
	return vm.chunkName
}

// CallerContext returns the current caller context for code loading.
func (vm *VM) CallerContext() *LuaCallerContext {
	return &LuaCallerContext{
		ScriptName: vm.chunkName,
		VMID:       vm.vmID,
		CallDepth:  len(vm.callStack),
	}
}

// SetIoProvider sets the IO provider for this VM.
func (vm *VM) SetIoProvider(provider LuaIoProvider) {
	vm.ioProvider = provider
}

// IoProvider returns the current IO provider, or nil if none is set.
func (vm *VM) IoProvider() LuaIoProvider {
	return vm.ioProvider
}

// SetOsProvider sets the OS provider for this VM.
func (vm *VM) SetOsProvider(provider LuaOsProvider) {
	vm.osProvider = provider
}

// OsProvider returns the current OS provider, or nil if none is set.
func (vm *VM) OsProvider() LuaOsProvider {
	return vm.osProvider
}

// SetDebugProvider sets the debug provider for this VM.
func (vm *VM) SetDebugProvider(provider LuaDebugProvider) {
	vm.debugProvider = provider
}

// DebugProvider returns the current debug provider, or nil if none is set.
func (vm *VM) DebugProvider() LuaDebugProvider {
	return vm.debugProvider
}

// SetContext sets the context for cooperative cancellation.
func (vm *VM) SetContext(ctx context.Context) {
	vm.ctx = ctx
}

// Context returns the current context, or nil if none is set.
func (vm *VM) Context() context.Context {
	return vm.ctx
}

// SetLimits sets execution limits on the VM.
func (vm *VM) SetLimits(limits Limits) {
	vm.limits = limits
}

// GetLimits returns the current execution limits.
func (vm *VM) GetLimits() Limits {
	return vm.limits
}

// InstructionCount returns the current checkpoint visit count.
func (vm *VM) InstructionCount() int64 {
	return vm.instrCount
}

// ResetInstructionCount resets the checkpoint visit counter to zero.
func (vm *VM) ResetInstructionCount() {
	vm.instrCount = 0
}
