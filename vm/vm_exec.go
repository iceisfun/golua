package vm

import (
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/iceisfun/golua/compiler"
)

// makeClosure instantiates subProto as a closure of the running frame, binding
// each upvalue either to a live stack slot or to one of the enclosing closure's
// upvalues. Kept out of the execute() dispatch switch to keep the switch body
// readable; OP_CLOSURE is rare enough that the call costs nothing. Lifting it
// does shrink execute(), but do not read a benchmark win into that -- see the
// code-placement caveat in PERF.md.
func (vm *VM) makeClosure(frame *callFrame, subProto *compiler.Proto) *Closure {
	// A single fresh stack capture (`function() return i end`) is by far the
	// most common closure shape, and the only one that can safely share an
	// allocation with its upvalue — see closureWithUpvalue.
	if len(subProto.Upvalues) == 1 && subProto.Upvalues[0].InStack {
		idx := frame.base + subProto.Upvalues[0].Index
		if existing := vm.findOpenUpvalue(idx); existing != nil {
			cl := NewClosure(subProto)
			cl.Upvalues[0] = existing
			return cl
		}
		cl, uv := newFusedClosure(vm, subProto, idx)
		vm.openUpvalues = append(vm.openUpvalues, uv)
		return cl
	}

	cl := NewClosure(subProto)
	for i, uvDesc := range subProto.Upvalues {
		if uvDesc.InStack {
			// Capture from current stack
			cl.Upvalues[i] = vm.findOrCreateUpvalue(frame.base + uvDesc.Index)
		} else {
			// Capture from enclosing closure's upvalues
			cl.Upvalues[i] = frame.closure.Upvalues[uvDesc.Index]
		}
	}
	return cl
}

// call invokes a closure with the given arguments and returns results.
func (vm *VM) call(closure *Closure, args []Value, nResults int) ([]Value, error) {
	vm.checkCallDepth()

	proto := closure.Proto

	// Set up the stack frame
	base := vm.top
	savedTop := vm.top
	vm.ensureStack(base + proto.MaxStack + stackSafetyMargin)

	// Handle varargs
	numParams := proto.NumParams
	numArgs := len(args)
	numVararg := 0
	var varargSlice []Value

	if proto.IsVarArg {
		// Copy fixed params first
		for i := 0; i < numParams && i < numArgs; i++ {
			vm.stack[base+i] = args[i]
		}
		// Nil-fill missing fixed params
		for i := numArgs; i < numParams; i++ {
			vm.stack[base+i] = Nil
		}
		// Store varargs in a Go slice to prevent cross-frame overlap.
		// Previously stored on the shared stack at base+MaxStack+256, but
		// callee frames could overwrite caller varargs.
		if numArgs > numParams {
			numVararg = numArgs - numParams
			varargSlice = make([]Value, numVararg)
			copy(varargSlice, args[numParams:numParams+numVararg])
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
		closure:               closure,
		funcValue:             NewFunction(closure),
		pc:                    0,
		base:                  base,
		nResults:              nResults,
		isVararg:              proto.IsVarArg,
		numVararg:             numVararg,
		varargs:               varargSlice,
		argc:                  UseVMTop, // Lua frames use vm.top for ArgCount
		ftransfer:             1,
		ntransfer:             min(numArgs, numParams),
		callName:              vm.pendingCallName,
		callNameWhat:          vm.pendingCallNameWhat,
		suppressTracebackName: vm.pendingSuppressTracebackName,
	}
	vm.pendingCallName = ""
	vm.pendingCallNameWhat = ""
	vm.pendingSuppressTracebackName = false
	vm.callStack = append(vm.callStack, frame)

	// Update vm.top to point past this frame's registers
	// This ensures nested calls get non-overlapping stack regions
	vm.top = base + proto.MaxStack

	// Fire call hook after frame is pushed
	vm.fireCallHook()

	// Save and restore lastHookLine/lastHookPC around the function call.
	// In Lua 5.4, L->oldpc is per-function state (effectively restored when
	// returning to the caller). Without this, the called function's line
	// tracking would interfere with the caller's.
	savedHookLine := vm.lastHookLine
	savedHookPC := vm.lastHookPC
	// Reset for the new function. VARARGPREP (pc=0) is handled specially
	// in execute() to suppress its own line hook.
	if vm.hookMask != 0 {
		vm.lastHookLine = -1
		vm.lastHookPC = 0
	}

	// Execute
	results, err := vm.execute()
	if err != nil {
		vm.lastHookLine = savedHookLine
		vm.lastHookPC = savedHookPC
		if le, ok := err.(*LuaError); ok {
			panic(le)
		}
		panic(&LuaError{Value: NewString(err.Error())})
	}

	// Restore lastHookLine/lastHookPC so the caller continues with its own line tracking
	vm.lastHookLine = savedHookLine
	vm.lastHookPC = savedHookPC

	// Pop call frame and restore top
	vm.callStack = vm.callStack[:len(vm.callStack)-1]
	vm.top = savedTop

	// Clear dead stack slots so Go's GC can collect objects
	// that were only reachable from the called function's frame.
	//
	// On error, defer clearing: ProtectedCall may need live stack values
	// for pending to-be-closed variables in this frame.
	if err == nil {
		clearEnd := base + proto.MaxStack
		if clearEnd > len(vm.stack) {
			clearEnd = len(vm.stack)
		}
		for i := vm.top; i < clearEnd; i++ {
			vm.stack[i] = Value{}
		}
	}

	return results, err
}

// CheckInterrupt checks for context cancellation and instruction limits.
// When neither ctx nor MaxInstructions is set, this is essentially free
// (two comparisons returning nil).
func (vm *VM) CheckInterrupt() error {
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
	if vm.limits.GCStepInterval > 0 {
		vm.gcStepCounter++
		if vm.gcStepCounter >= vm.limits.GCStepInterval {
			vm.gcStepCounter = 0
			// Clear dead stack slots above vm.top to help Go's GC
			// collect objects that are only reachable from dead registers.
			for ci := vm.top; ci < len(vm.stack); ci++ {
				vm.stack[ci] = Nil
			}
			runtime.GC()
			runtime.GC()
			vm.processGcFinalizersOnly()
		}
	}
	return nil
}

// execute runs the current call frame until it returns.
func (vm *VM) execute() ([]Value, error) {
	frame := &vm.callStack[len(vm.callStack)-1]
	proto := frame.closure.Proto
	code := proto.Code
	consts := frame.closure.ConstValues()

	for {
		frame = &vm.callStack[len(vm.callStack)-1]

		if frame.pc >= len(code) {
			return nil, nil
		}

		inst := code[frame.pc]
		frame.pc++

		// Hook dispatch: line and count hooks (fast path: hookMask == 0 skips entirely)
		// Skip line hook for VARARGPREP (pc=0) — in Lua 5.4, OP_VARARGPREP
		// suppresses its own line hook and sets L->oldpc = 1 so the next
		// instruction fires. Count hooks still fire for VARARGPREP.
		if vm.hookMask != 0 && !vm.inHook {
			skipLine := frame.pc == 1 && inst.OpCode() == compiler.OP_VARARGPREP
			if skipLine {
				// Only fire count hook, not line hook, for VARARGPREP
				if vm.hookMask&HookMaskCount != 0 {
					vm.hookCounter--
					if vm.hookCounter <= 0 {
						vm.hookCounter = vm.hookCount
						line := 0
						if len(proto.Lines) > 0 {
							line = proto.Lines[0]
						}
						vm.fireHook(hookEventCount, line)
						frame = &vm.callStack[len(vm.callStack)-1]
					}
				}
			} else if vm.checkLineCountHooks(proto, frame.pc-1) {
				// Re-fetch after hook callback may have modified call stack
				frame = &vm.callStack[len(vm.callStack)-1]
			}
		}

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
			vm.stack[frame.base+a] = consts[bx]

		case compiler.OP_LOADKX:
			a := inst.A()
			// Next instruction is EXTRAARG with the constant index
			extra := code[frame.pc]
			frame.pc++
			ax := extra.Ax()
			vm.stack[frame.base+a] = consts[ax]

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
			key := consts[c]
			if t := table.AsTable(); t != nil {
				val, err := vm.tableGet(t, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
			} else if mm := vm.getMetafield(table, MetaIndex); !mm.IsNil() {
				val, err := vm.resolveIndex(mm, table, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
			} else {
				uvName := ""
				if b < len(proto.Upvalues) {
					uvName = proto.Upvalues[b].Name
				}
				if uvName != "" {
					return nil, vm.runtimeError("attempt to index a %s value (upvalue '%s')", vm.ObjTypeName(table), uvName)
				}
				return nil, vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(table))
			}

		case compiler.OP_GETTABLE:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			key := vm.stack[frame.base+c]
			if ct, ok := table.ptr.(*Table); ok && table.typ == typeTable && !ct.isThread {
				if ct.metatable == nil {
					vm.stack[frame.base+a] = ct.Get(key)
				} else {
					val, err := vm.tableGet(table.ptr.(LuaTable), key)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = val
				}
			} else if mm := vm.getMetafield(table, MetaIndex); !mm.IsNil() {
				// Type metatable __index
				val, err := vm.resolveIndex(mm, table, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
			}

		case compiler.OP_GETI:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			if ct, ok := table.ptr.(*Table); ok && table.typ == typeTable && !ct.isThread {
				if ct.metatable == nil {
					vm.stack[frame.base+a] = ct.GetInt(c)
				} else {
					val, err := vm.tableGetInt(table.ptr.(LuaTable), c)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = val
				}
			} else if mm := vm.getMetafield(table, MetaIndex); !mm.IsNil() {
				val, err := vm.resolveIndex(mm, table, NewInt(int64(c)))
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
			}

		case compiler.OP_GETFIELD:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			key := proto.Constants[c].SVal
			if ct, ok := table.ptr.(*Table); ok && table.typ == typeTable && !ct.isThread {
				// Fast path: concrete *Table with no metatable (avoids interface assertions)
				if ct.metatable == nil {
					vm.stack[frame.base+a] = ct.GetString(key)
				} else {
					val, err := vm.tableGetString(table.ptr.(LuaTable), key)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = val
				}
			} else if mm := vm.getMetafield(table, MetaIndex); !mm.IsNil() {
				val, err := vm.resolveIndex(mm, table, NewString(key))
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
			}

		case compiler.OP_SETTABUP:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := frame.closure.Upvalues[a].Get()
			key := consts[b]
			var value Value
			if inst.K() != 0 {
				value = consts[c]
			} else {
				value = vm.stack[frame.base+c]
			}
			if t := table.AsTable(); t != nil {
				if err := vm.tableSet(t, key, value); err != nil {
					return nil, err
				}
			} else {
				uvName := ""
				if a < len(proto.Upvalues) {
					uvName = proto.Upvalues[a].Name
				}
				if uvName != "" {
					return nil, vm.runtimeError("attempt to index a %s value (upvalue '%s')", vm.ObjTypeName(table), uvName)
				}
				return nil, vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(table))
			}

		case compiler.OP_SETTABLE:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+a]
			key := vm.stack[frame.base+b]
			var value Value
			if inst.K() != 0 {
				value = consts[c]
			} else {
				value = vm.stack[frame.base+c]
			}
			if ct, ok := table.ptr.(*Table); ok && table.typ == typeTable && !ct.isThread {
				if ct.metatable == nil {
					if err := ct.Set(key, value); err != nil {
						return nil, vm.runtimeError("%s", err)
					}
				} else if err := vm.tableSet(table.ptr.(LuaTable), key, value); err != nil {
					return nil, err
				}
			} else if mm := vm.getMetafield(table, MetaNewIndex); !mm.IsNil() {
				if mm.IsFunction() || mm.IsNativeFunc() {
					_, err := vm.callMetamethod3("newindex", mm, table, key, value)
					if err != nil {
						return nil, err
					}
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				} else if mm.IsTable() {
					if tbl, ok := mm.AsTable().(*Table); ok && tbl.IsThread() {
						if err := vm.newIndexValue(mm, key, value, vm.MaxMetaDepth()); err != nil {
							return nil, err
						}
						frame = &vm.callStack[len(vm.callStack)-1]
						proto = frame.closure.Proto
						code = proto.Code
						consts = frame.closure.ConstValues()
					} else {
						if err := vm.tableSet(mm.AsTable(), key, value); err != nil {
							return nil, err
						}
					}
				} else {
					if err := vm.newIndexValue(mm, key, value, vm.MaxMetaDepth()); err != nil {
						return nil, err
					}
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				}
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(a))
			}

		case compiler.OP_SETI:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+a]
			var value Value
			if inst.K() != 0 {
				value = consts[c]
			} else {
				value = vm.stack[frame.base+c]
			}
			if ct, ok := table.ptr.(*Table); ok && table.typ == typeTable && !ct.isThread {
				if ct.metatable == nil {
					ct.SetInt(b, value)
				} else {
					if err := vm.tableSetInt(table.ptr.(LuaTable), b, value); err != nil {
						return nil, err
					}
				}
			} else if mm := vm.getMetafield(table, MetaNewIndex); !mm.IsNil() {
				if mm.IsFunction() || mm.IsNativeFunc() {
					_, err := vm.callMetamethod3("newindex", mm, table, NewInt(int64(b)), value)
					if err != nil {
						return nil, err
					}
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				} else if mm.IsTable() {
					if tbl, ok := mm.AsTable().(*Table); ok && tbl.IsThread() {
						if err := vm.newIndexValue(mm, NewInt(int64(b)), value, vm.MaxMetaDepth()); err != nil {
							return nil, err
						}
						frame = &vm.callStack[len(vm.callStack)-1]
						proto = frame.closure.Proto
						code = proto.Code
						consts = frame.closure.ConstValues()
					} else {
						if err := vm.tableSet(mm.AsTable(), NewInt(int64(b)), value); err != nil {
							return nil, err
						}
					}
				} else {
					if err := vm.newIndexValue(mm, NewInt(int64(b)), value, vm.MaxMetaDepth()); err != nil {
						return nil, err
					}
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				}
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(a))
			}

		case compiler.OP_SETFIELD:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+a]
			key := proto.Constants[b].SVal
			var value Value
			if inst.K() != 0 {
				value = consts[c]
			} else {
				value = vm.stack[frame.base+c]
			}
			if ct, ok := table.ptr.(*Table); ok && table.typ == typeTable && !ct.isThread {
				// Fast path: concrete *Table with no metatable. Pass the
				// cached constant Value so a first-time key needs no boxing.
				if ct.metatable == nil {
					ct.setStringValue(consts[b], value)
				} else {
					if err := vm.tableSetString(table.ptr.(LuaTable), key, value); err != nil {
						return nil, err
					}
				}
			} else if mm := vm.getMetafield(table, MetaNewIndex); !mm.IsNil() {
				if mm.IsFunction() || mm.IsNativeFunc() {
					_, err := vm.callMetamethod3("newindex", mm, table, NewString(key), value)
					if err != nil {
						return nil, err
					}
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				} else if mm.IsTable() {
					if tbl, ok := mm.AsTable().(*Table); ok && tbl.IsThread() {
						if err := vm.newIndexValue(mm, NewString(key), value, vm.MaxMetaDepth()); err != nil {
							return nil, err
						}
						frame = &vm.callStack[len(vm.callStack)-1]
						proto = frame.closure.Proto
						code = proto.Code
						consts = frame.closure.ConstValues()
					} else {
						if err := vm.tableSet(mm.AsTable(), NewString(key), value); err != nil {
							return nil, err
						}
					}
				} else {
					if err := vm.newIndexValue(mm, NewString(key), value, vm.MaxMetaDepth()); err != nil {
						return nil, err
					}
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				}
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(a))
			}

		case compiler.OP_NEWTABLE:
			a := inst.A()
			// IvABC: vB (bits 16-21, 6 bits) = hash log, vC (bits 22-31, 10 bits) = array size
			vB := inst.VB()
			vC := inst.VC()
			if inst.K() != 0 {
				// Array size in next EXTRAARG instruction
				frame.pc++
				vC = code[frame.pc-1].Ax()
			}
			nHash := 0
			if vB > 0 {
				nHash = 1 << (vB - 1)
			}
			if vC > 0 || nHash > 0 {
				vm.stack[frame.base+a] = NewTable(NewTableWithSize(vC, nHash))
			} else {
				vm.stack[frame.base+a] = NewTable(NewEmptyTable())
			}

		case compiler.OP_SELF:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := vm.stack[frame.base+b]
			vm.stack[frame.base+a+1] = table
			key := proto.Constants[c].SVal
			if t := table.AsTable(); t != nil {
				val, err := vm.tableGetString(t, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
			} else if table.IsString() && vm.stringMeta != nil {
				// String method call - resolve through __index metamethod only
				idx := vm.stringMeta.Get(metaIndex)
				var val Value
				if idx.IsTable() {
					val, _ = vm.tableGetString(idx.AsTable(), key)
				} else if idx.IsFunction() || idx.IsNativeFunc() {
					var err error
					val, err = vm.callMetamethod("index", idx, table, NewString(key))
					if err != nil {
						return nil, err
					}
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				}
				vm.stack[frame.base+a] = val
			} else if ud := table.AsUserdata(); ud != nil {
				// Userdata method call - use __index from userdata metatable
				if mt := ud.Metatable(); mt != nil {
					index := mt.Get(metaIndex)
					if index.IsTable() {
						val, err := vm.tableGetString(index.AsTable(), key)
						if err != nil {
							return nil, err
						}
						vm.stack[frame.base+a] = val
					} else if index.IsFunction() || index.IsNativeFunc() {
						val, err := vm.callMetamethod("index", index, table, NewString(key))
						if err != nil {
							return nil, err
						}
						vm.stack[frame.base+a] = val
						frame = &vm.callStack[len(vm.callStack)-1]
						proto = frame.closure.Proto
						code = proto.Code
						consts = frame.closure.ConstValues()
					} else {
						return nil, vm.runtimeError("attempt to index a userdata value")
					}
				} else {
					return nil, vm.runtimeError("attempt to index a userdata value")
				}
			} else if mm := vm.getMetafield(table, MetaIndex); !mm.IsNil() {
				val, err := vm.resolveIndex(mm, table, NewString(key))
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
			}

		case compiler.OP_ADDI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			if v.IsInt() {
				vm.stack[frame.base+a] = NewInt(v.AsInt() + int64(sc))
			} else if v.IsFloat() {
				vm.stack[frame.base+a] = NewFloat(v.fval() + float64(sc))
			} else {
				// Non-numeric: dispatch to the metamethod implied by the
				// following MMBINI. The compiler emits ADDI for both x + n
				// (TM_ADD) and the rewrite x - n → x + (-n) (TM_SUB), so the
				// tag must come from MMBINI rather than being hardcoded. The
				// MMBINI carries the user-written immediate (un-negated) in
				// its sB field — that's the value the metamethod must see.
				mmName := MetaAdd
				immForMM := int64(sc)
				flip := false
				if frame.pc < len(code) {
					nextInst := code[frame.pc]
					if nextInst.OpCode() == compiler.OP_MMBINI {
						mmName = decodeBytecodeMetamethodTag(nextInst.C()).String()
						immForMM = int64(nextInst.SB())
						flip = nextInst.K() == 1
					}
				}
				immVal := NewInt(immForMM)
				arg1, arg2 := v, immVal
				if flip {
					arg1, arg2 = immVal, v
				}
				if mm := vm.getArithMetamethod(arg1, arg2, mmName); !mm.IsNil() {
					result, err := vm.callMetamethod(MetaEvent(mmName), mm, arg1, arg2)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = result
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
					// ADDI has fully handled the arithmetic metamethod (result
					// written to the destination register a). Skip the follow-up
					// OP_MMBINI so it does not invoke the metamethod a second
					// time — its operand register still holds the original
					// non-numeric value, so its skip guard would not fire.
					// (Bitwise-immediate ops SHLI/SHRI do NOT pre-handle their
					// metamethod and correctly fall through to MMBINI.)
					if frame.pc < len(code) && code[frame.pc].OpCode() == compiler.OP_MMBINI {
						frame.pc++
					}
				} else {
					return nil, vm.runtimeError("attempt to perform arithmetic on a %s value%s", vm.ObjTypeName(v), vm.varInfo(b))
				}
			}

		case compiler.OP_ADDK, compiler.OP_SUBK, compiler.OP_MULK, compiler.OP_MODK,
			compiler.OP_POWK, compiler.OP_DIVK, compiler.OP_IDIVK:
			a, b, c := inst.A(), inst.B(), inst.C()
			v := vm.stack[frame.base+b]
			kv := consts[c]
			result, err := vm.arithK(op, v, kv, b)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_BANDK, compiler.OP_BORK, compiler.OP_BXORK:
			a, b, c := inst.A(), inst.B(), inst.C()
			v := vm.stack[frame.base+b]
			kv := consts[c]
			result, err := vm.bitwiseK(op, v, kv, b)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_SHLI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			if !v.IsString() {
				if i, ok := v.ToInt(); ok {
					if sc < 0 {
						vm.stack[frame.base+a] = NewInt(i << uint(-sc))
					} else {
						vm.stack[frame.base+a] = NewInt(int64(uint64(i) >> uint(sc)))
					}
				} else if v.IsNumber() {
					return nil, vm.runtimeErrorForNumber(b)
				}
			}

		case compiler.OP_SHRI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			if !v.IsString() {
				if i, ok := v.ToInt(); ok {
					vm.stack[frame.base+a] = NewInt(int64(sc) << uint(i))
				} else if v.IsNumber() {
					return nil, vm.runtimeErrorForNumber(b)
				}
			}

		case compiler.OP_ADD, compiler.OP_SUB, compiler.OP_MUL, compiler.OP_MOD,
			compiler.OP_POW, compiler.OP_DIV, compiler.OP_IDIV:
			a, b, c := inst.A(), inst.B(), inst.C()
			v1 := vm.stack[frame.base+b]
			v2 := vm.stack[frame.base+c]
			// Inline the number fast paths to avoid passing two 32-byte Value
			// structs by value to arith() on the hot arithmetic path. These
			// mirror arith()'s fast paths exactly; mixed-number, string,
			// metamethod and error reporting fall through to vm.arith.
			if v1.typ == typeFloat && v2.typ == typeFloat {
				n1, n2 := v1.fval(), v2.fval()
				var r float64
				switch op {
				case compiler.OP_ADD:
					r = n1 + n2
				case compiler.OP_SUB:
					r = n1 - n2
				case compiler.OP_MUL:
					r = n1 * n2
				case compiler.OP_DIV:
					r = n1 / n2
				case compiler.OP_IDIV:
					r = math.Floor(n1 / n2)
				case compiler.OP_MOD:
					r = luaNumMod(n1, n2)
				default: // OP_POW
					r = PowWithSubnormalFix(n1, n2)
				}
				vm.stack[frame.base+a] = NewFloat(r)
			} else if v1.typ == typeInt && v2.typ == typeInt && op != compiler.OP_DIV && op != compiler.OP_POW {
				i1, i2 := v1.ival(), v2.ival()
				var r int64
				switch op {
				case compiler.OP_ADD:
					r = i1 + i2
				case compiler.OP_SUB:
					r = i1 - i2
				case compiler.OP_MUL:
					r = i1 * i2
				case compiler.OP_IDIV:
					if i2 == 0 {
						return nil, vm.runtimeError("attempt to divide by zero")
					}
					if i2 == -1 {
						r = -i1 // avoid MinInt64/-1 overflow panic
					} else {
						r = i1 / i2
						if (i1^i2) < 0 && r*i2 != i1 {
							r-- // floor toward negative infinity
						}
					}
				default: // OP_MOD
					if i2 == 0 {
						return nil, vm.runtimeError("attempt to perform 'n%%0'")
					}
					if i2 == -1 {
						r = 0
					} else {
						r = i1 % i2
						if r != 0 && (r^i2) < 0 {
							r += i2
						}
					}
				}
				vm.stack[frame.base+a] = NewInt(r)
			} else {
				result, err := vm.arith(op, v1, v2, b, c)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = result
			}

		case compiler.OP_BAND, compiler.OP_BOR, compiler.OP_BXOR,
			compiler.OP_SHL, compiler.OP_SHR:
			a, b, c := inst.A(), inst.B(), inst.C()
			v1 := vm.stack[frame.base+b]
			v2 := vm.stack[frame.base+c]
			result, err := vm.bitwise(op, v1, v2, b, c)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_MMBIN, compiler.OP_MMBINI, compiler.OP_MMBINK:
			if op == compiler.OP_MMBINI {
				a := inst.A()
				sb := inst.SB()
				tag := decodeBytecodeMetamethodTag(inst.C())
				left := vm.stack[frame.base+a]
				right := NewInt(int64(sb))
				if inst.K() == 1 {
					left, right = right, left
				}
				skip := false
				switch tag {
				case compiler.TM_ADD, compiler.TM_SUB, compiler.TM_MUL, compiler.TM_MOD, compiler.TM_POW, compiler.TM_DIV, compiler.TM_IDIV:
					skip = left.IsNumber() && right.IsNumber()
				case compiler.TM_BAND, compiler.TM_BOR, compiler.TM_BXOR, compiler.TM_SHL, compiler.TM_SHR:
					_, ok1 := left.ToInt()
					_, ok2 := right.ToInt()
					skip = !left.IsString() && !right.IsString() && ok1 && ok2
				}
				if skip {
					break
				}
				mmName := tag.String()
				mm := vm.getArithMetamethod(left, right, mmName)
				if mm.IsNil() {
					if tag == compiler.TM_SHL || tag == compiler.TM_SHR || tag == compiler.TM_BAND || tag == compiler.TM_BOR || tag == compiler.TM_BXOR {
						if !left.IsNumber() {
							return nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(left), vm.varInfo(a))
						}
						return nil, vm.runtimeError("attempt to perform bitwise operation on a %s value", vm.ObjTypeName(right))
					}
					if !left.IsNumber() {
						return nil, vm.runtimeError("attempt to perform arithmetic on a %s value%s", vm.ObjTypeName(left), vm.varInfo(a))
					}
					return nil, vm.runtimeError("attempt to perform arithmetic on a %s value", vm.ObjTypeName(right))
				}
				result, err := vm.callMetamethod(MetaEvent(mmName), mm, left, right)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = result
			}

		case compiler.OP_UNM:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]
			if v.IsNumber() {
				if v.IsInt() {
					vm.stack[frame.base+a] = NewInt(-v.AsInt())
				} else {
					vm.stack[frame.base+a] = NewFloat(-v.AsFloat())
				}
			} else if mm := vm.getMetafield(v, MetaUnm); !mm.IsNil() {
				result, err := vm.callMetamethod(MetaEvent(MetaUnm), mm, v, v)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = result
			} else {
				return nil, vm.runtimeError("attempt to perform arithmetic on a %s value%s", vm.ObjTypeName(v), vm.varInfo(b))
			}

		case compiler.OP_BNOT:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]
			// Lua 5.4: bitwise ops do NOT coerce strings
			done := false
			if !v.IsString() {
				if i, ok := v.ToInt(); ok {
					vm.stack[frame.base+a] = NewInt(^i)
					done = true
				}
			}
			if !done {
				if mm := vm.getMetafield(v, MetaBNot); !mm.IsNil() {
					result, err := vm.callMetamethod(MetaEvent(MetaBNot), mm, v, v)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = result
				} else if v.IsNumber() {
					return nil, vm.runtimeErrorForNumber(b)
				} else {
					return nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(v), vm.varInfo(b))
				}
			}

		case compiler.OP_NOT:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]
			vm.stack[frame.base+a] = NewBool(!v.ToBool())

		case compiler.OP_LEN:
			a, b := inst.A(), inst.B()
			v := vm.stack[frame.base+b]

			if v.IsString() {
				vm.stack[frame.base+a] = NewInt(int64(len(v.AsString())))
			} else {
				// Check for __len metamethod
				op := MetaLen
				mm := vm.getMetafield(v, op)
				if !mm.IsNil() {
					res, err := vm.callMetamethod(MetaEvent(MetaLen), mm, v, v)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = res
				} else if v.IsTable() && !v.isThread() {
					vm.stack[frame.base+a] = NewInt(int64(v.AsTable().Len()))
				} else {
					return nil, vm.runtimeError("attempt to get length of a %s value%s", vm.ObjTypeName(v), vm.varInfo(b))
				}
			}

		case compiler.OP_CONCAT:
			a, b := inst.A(), inst.B()

			// Fast path: 2-operand string concat (most common case: s = s .. "x")
			if b == 2 {
				v1 := vm.stack[frame.base+a]
				v2 := vm.stack[frame.base+a+1]
				if v1.typ == typeString && v2.typ == typeString {
					s1 := v1.ptr.(string)
					s2 := v2.ptr.(string)
					// Guard the result size before allocating, exactly like the
					// multi-operand path below. Without this, an unbounded concat
					// (e.g. `s = s .. s` doubling) drives the result past what Go
					// can allocate and triggers an UNCATCHABLE runtime fatal OOM
					// that aborts the host process — a sandbox escape. The guard
					// turns it into a catchable Lua error.
					if len(s1) > (1<<30)-len(s2) {
						return nil, vm.runtimeError("string length overflow")
					}
					vm.stack[frame.base+a] = NewString(s1 + s2)
					break
				}
			}

			// Concatenate b values starting at R[A] which ends at R[A+b-1]
			// The original implementation was building a string directly.
			// To support __concat, we must check if simplification is possible.

			// Optimization: Check if all are string/number
			allStringOrNum := true
			totalLen := 0
			for i := 0; i < b; i++ {
				v := vm.stack[frame.base+a+i]
				if !v.IsString() && !v.IsNumber() {
					allStringOrNum = false
					break
				}
				if v.IsString() {
					l := len(v.AsString())
					if totalLen > (1<<30)-l {
						return nil, vm.runtimeError("string length overflow")
					}
					totalLen += l
				} else {
					if totalLen > (1<<30)-20 {
						return nil, vm.runtimeError("string length overflow")
					}
					totalLen += 20
				}
			}

			if allStringOrNum {
				var builder strings.Builder
				builder.Grow(totalLen)
				for i := 0; i < b; i++ {
					v := vm.stack[frame.base+a+i]
					if v.IsString() {
						builder.WriteString(v.AsString())
					} else {
						builder.WriteString(v.String())
					}
				}
				vm.stack[frame.base+a] = NewString(builder.String())
			} else {
				// Fallback to pairwise concatenation to support __concat logic
				// Lua semantics: concat A..B..C is A..(B..C).
				// In lvm.c luaV_concat: "Lift semantic: the concatenation is performed from the last element to the first."

				// So loop from b-2 down to 0
				// buffer is at R[A + i] .. R[A+i+1]

				// Start with the last element
				if b >= 2 {
					current := vm.stack[frame.base+a+b-1]
					for i := b - 2; i >= 0; i-- {
						prev := vm.stack[frame.base+a+i]
						res, err := vm.concat(prev, current, a+i)
						if err != nil {
							return nil, err
						}
						current = res
					}
					vm.stack[frame.base+a] = current
				}
			}

		case compiler.OP_CLOSE:
			a := inst.A()
			vm.closeUpvalues(frame.base + a)

		case compiler.OP_TBC:
			// Mark variable as to-be-closed
			a := inst.A()
			val := vm.stack[frame.base+a]
			// Validate: nil and false are always OK; otherwise must have __close
			if !val.IsNil() && !(val.IsBool() && !val.AsBool()) {
				needErr := false
				if val.IsTable() {
					mt := val.AsTable().Metatable()
					if mt == nil {
						// Check type-level metatable (threads have no per-instance metatable)
						mt = vm.GetTypeMeta(val)
					}
					if mt == nil || mt.Get(metaClose).IsNil() {
						needErr = true
					}
				} else if ud := val.AsUserdata(); ud != nil {
					mt := ud.Metatable()
					if mt == nil || mt.Get(metaClose).IsNil() {
						needErr = true
					}
				} else {
					// Check type-level metatable (e.g. debug.setmetatable(0, {__close=...}))
					typeMT := vm.GetTypeMeta(val)
					if typeMT == nil || typeMT.Get(metaClose).IsNil() {
						needErr = true
					}
				}
				if needErr {
					varName := localName(frame.closure.Proto, a, frame.pc)
					return nil, vm.runtimeError("variable '%s' got a non-closable value", varName)
				}
			}
			vm.tbcVars = append(vm.tbcVars, frame.base+a)

		case compiler.OP_JMP:
			sj := inst.SJ()
			if sj < 0 {
				if err := vm.CheckInterrupt(); err != nil {
					return nil, err
				}
			}
			frame.pc += sj

		case compiler.OP_EQ:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := vm.stack[frame.base+b]
			eq, err := vm.equal(v1, v2)
			if err != nil {
				return nil, err
			}
			frame = &vm.callStack[len(vm.callStack)-1]
			if eq != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LT:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := vm.stack[frame.base+b]
			lt, err := vm.lessThan(v1, v2)
			if err != nil {
				return nil, err
			}
			frame = &vm.callStack[len(vm.callStack)-1]
			if lt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LE:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := vm.stack[frame.base+b]
			le, err := vm.lessEqual(v1, v2)
			if err != nil {
				return nil, err
			}
			frame = &vm.callStack[len(vm.callStack)-1]
			if le != (k == 1) {
				frame.pc++
			}

		case compiler.OP_EQK:
			a, b, k := inst.A(), inst.B(), inst.K()
			v1 := vm.stack[frame.base+a]
			v2 := consts[b]
			eq, err := vm.equal(v1, v2)
			if err != nil {
				return nil, err
			}
			frame = &vm.callStack[len(vm.callStack)-1]
			if eq != (k == 1) {
				frame.pc++
			}

		case compiler.OP_EQI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v1 := vm.stack[frame.base+a]
			// Create temp value for rapid comparison
			var v2 Value
			if v1.IsInt() {
				v2 = NewInt(int64(sb))
			} else {
				v2 = NewFloat(float64(sb))
			}
			eq, err := vm.equal(v1, v2)
			if err != nil {
				return nil, err
			}
			frame = &vm.callStack[len(vm.callStack)-1]
			if eq != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LTI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			lt, err := vm.lessThan(v, NewInt(int64(sb)))
			if err != nil {
				return nil, err
			}
			if lt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LEI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			le, err := vm.lessEqual(v, NewInt(int64(sb)))
			if err != nil {
				return nil, err
			}
			if le != (k == 1) {
				frame.pc++
			}

		case compiler.OP_GTI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			gt, err := vm.lessThan(NewInt(int64(sb)), v)
			if err != nil {
				return nil, err
			}
			if gt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_GEI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			ge, err := vm.lessEqual(NewInt(int64(sb)), v)
			if err != nil {
				return nil, err
			}
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
			if err := vm.CheckInterrupt(); err != nil {
				return nil, err
			}
			a, b, c := inst.A(), inst.B(), inst.C()
			if err := vm.doCall(frame, a, b, c); err != nil {
				return nil, err
			}

		case compiler.OP_TAILCALL:
			if err := vm.CheckInterrupt(); err != nil {
				return nil, err
			}
			a, b, _ := inst.A(), inst.B(), inst.C()
			// Save proto/PC before tail call rewrites the frame, so we can
			// produce a good error message if the callee turns out to be
			// non-callable.
			tailProto := proto
			tailPC := frame.pc - 1 // PC of the TAILCALL instruction
			// Tail call optimization - reuse current frame
			fn := vm.stack[frame.base+a]

			// Collect arguments. args lives only within this handler (consumed
			// by frame setup or the native tail-call path), so a stack-local
			// buffer for the small-arg-count case avoids a per-tailcall heap
			// allocation. The __call dispatch loop reassigns args to a heap
			// slice before re-entering, so this buffer is not aliased there.
			var tailArgBuf [8]Value
			var args []Value
			var nTailArgs int
			if b == 0 {
				nTailArgs = vm.top - (frame.base + a + 1)
			} else {
				nTailArgs = b - 1
			}
			if nTailArgs <= len(tailArgBuf) {
				args = tailArgBuf[:nTailArgs]
			} else {
				args = make([]Value, nTailArgs)
			}
			if b == 0 {
				copy(args, vm.stack[frame.base+a+1:vm.top])
			} else {
				copy(args, vm.stack[frame.base+a+1:frame.base+a+b])
			}

			// Close upvalues
			vm.closeUpvalues(frame.base)

			// Dispatch loop for __call support
			for {
				if fn.IsFunction() {
					closure := fn.AsClosure()
					// Reuse the frame
					frame.closure = closure
					frame.funcValue = fn
					frame.pc = 0
					frame.isTailCall = true
					frame.argc = UseVMTop // Lua frame: use vm.top for ArgCount
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
							frame.numVararg = numArgs - numParams
							frame.varargs = make([]Value, frame.numVararg)
							copy(frame.varargs, args[numParams:numParams+frame.numVararg])
						} else {
							frame.numVararg = 0
							frame.varargs = nil
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

					// Set transfer info for the tail call target so call hooks
					// see the correct arguments via ftransfer/ntransfer.
					frame.ftransfer = 1
					frame.ntransfer = min(numArgs, numParams)

					// Fire tail call hook AFTER frame is updated with new closure
					// and parameters are on the stack, so debug.getinfo and
					// debug.getlocal see the target function and its arguments.
					vm.fireTailCallHook()
					break // Continue outer loop (instruction loop)
				} else if fn.IsNativeFunc() {
					// Native function tail call - can't truly optimize, just call.
					// We must push a proper native frame so that stack-walking
					// functions (e.g. error()'s GetSourceLocation) can correctly
					// skip this frame, matching Lua 5.4's C-function tail call
					// behavior where the C frame exists on the call stack.
					//
					// Place the native frame at vm.top (NOT frame.base) to avoid
					// overwriting the caller's local variables. This matches Lua
					// 5.4 which doesn't reuse the Lua frame for C tail calls,
					// so debug.getlocal on the caller still sees correct values.
					nf := fn.AsNativeFunc()
					nativeBase := vm.top
					vm.ensureStack(nativeBase + 1 + len(args) + 4)
					vm.stack[nativeBase] = fn
					for i, arg := range args {
						vm.stack[nativeBase+1+i] = arg
					}
					// Clear slots beyond arguments
					clearStart := nativeBase + 1 + len(args)
					clearEnd := clearStart + 4
					if clearEnd > len(vm.stack) {
						clearEnd = len(vm.stack)
					}
					for i := clearStart; i < clearEnd; i++ {
						vm.stack[i] = Nil
					}
					// Advance vm.top past the native frame's arguments so nested
					// calls (e.g. ProtectedCall inside pcall) allocate frames
					// that don't overlap with the native frame's stack region.
					savedTop := vm.top
					vm.top = nativeBase + 1 + len(args)
					// Push native call frame
					nativeFrame := callFrame{
						base:      nativeBase,
						argc:      len(args),
						funcValue: fn,
					}
					vm.callStack = append(vm.callStack, nativeFrame)
					nResults := nf(vm)
					vm.callStack = vm.callStack[:len(vm.callStack)-1]
					vm.top = savedTop
					var results []Value
					if nResults <= len(vm.retBuf) {
						copy(vm.retBuf[:nResults], vm.stack[nativeBase:nativeBase+nResults])
						results = vm.retBuf[:nResults]
					} else {
						results = make([]Value, nResults)
						copy(results, vm.stack[nativeBase:nativeBase+nResults])
					}
					return results, nil
				} else {
					// Check for __call metamethod
					op := MetaCall
					mm := vm.getMetafield(fn, op)
					if !mm.IsNil() {
						// Create new args with self (fn) prepended
						newArgs := make([]Value, len(args)+1)
						newArgs[0] = fn
						copy(newArgs[1:], args)
						args = newArgs
						fn = mm
						continue // Retry dispatch with metamethod
					}
					vi := ""
					name, what := regObjName(tailProto, tailPC, a)
					if name != "" {
						vi = fmt.Sprintf(" (%s '%s')", what, name)
					}
					return nil, vm.runtimeError("attempt to call a %s value%s", vm.ObjTypeName(fn), vi)
				}
			}
			if fn.IsFunction() {
				// Tailcall replaced the closure; refresh proto/code/consts
				proto = frame.closure.Proto
				code = proto.Code
				consts = frame.closure.ConstValues()
				continue
			}

		case compiler.OP_RETURN:
			a, b, c := inst.A(), inst.B(), inst.C()
			_ = c // c contains info about closing upvalues

			// Collect return values BEFORE closing upvalues, since closeUpvalues
			// may modify the stack (running __close metamethods).
			// Use a stack-local buffer to survive __close clobbering vm.retBuf.
			var nret int
			if b == 0 {
				nret = vm.top - (frame.base + a)
			} else {
				nret = b - 1
			}
			var localBuf [8]Value
			var saved []Value
			if nret <= len(localBuf) {
				saved = localBuf[:nret]
			} else {
				saved = make([]Value, nret)
			}
			copy(saved, vm.stack[frame.base+a:frame.base+a+nret])

			// Close upvalues and run __close metamethods BEFORE the return hook.
			// Lua 5.4 runs __close before the return hook for the function itself.
			vm.closeUpvalues(frame.base)

			// Fire return hook after __close metamethods
			frame.ftransfer = a + 1
			frame.ntransfer = nret
			vm.fireReturnHook()

			// Copy to vm.retBuf AFTER close handlers have finished
			if nret <= len(vm.retBuf) {
				copy(vm.retBuf[:nret], saved)
				return vm.retBuf[:nret], nil
			}
			return saved, nil

		case compiler.OP_RETURN0:
			vm.closeUpvalues(frame.base)
			frame.ftransfer = 0
			frame.ntransfer = 0
			vm.fireReturnHook()
			return nil, nil

		case compiler.OP_RETURN1:
			a := inst.A()
			result := vm.stack[frame.base+a]
			vm.closeUpvalues(frame.base)
			frame.ftransfer = a + 1
			frame.ntransfer = 1
			vm.fireReturnHook()
			vm.retBuf[0] = result
			return vm.retBuf[:1], nil

		case compiler.OP_FORLOOP:
			a, bx := inst.A(), inst.Bx()
			// R[A] = index, R[A+1] = counter (remaining iterations), R[A+2] = step
			stepVal := vm.stack[frame.base+a+2]
			if stepVal.IsInt() {
				// Integer for loop: counter-based (Lua 5.4 semantics)
				// R[A+1] is the remaining iterations counter, decremented each loop.
				// Use uint64 arithmetic: when the total iteration count exceeds
				// MaxInt64, the counter is stored as a negative int64 but must
				// be treated as unsigned for correct wrap-around behavior.
				ucounter := uint64(vm.stack[frame.base+a+1].AsInt())
				ucounter--
				vm.stack[frame.base+a+1] = NewInt(int64(ucounter))
				if ucounter != ^uint64(0) { // pre-decrement was > 0 (unsigned)
					idx := vm.stack[frame.base+a].AsInt()
					step := stepVal.AsInt()
					idx += step
					vm.stack[frame.base+a] = NewInt(idx)
					vm.stack[frame.base+a+3] = NewInt(idx)
					if err := vm.CheckInterrupt(); err != nil {
						return nil, err
					}
					frame.pc -= bx + 1
				}
			} else {
				// Float for loop
				idx := vm.stack[frame.base+a].AsFloat()
				limit := vm.stack[frame.base+a+1].AsFloat()
				step := stepVal.AsFloat()
				idx += step
				vm.stack[frame.base+a] = NewFloat(idx)
				if step >= 0 {
					if idx <= limit {
						if err := vm.CheckInterrupt(); err != nil {
							return nil, err
						}
						frame.pc -= bx + 1
						vm.stack[frame.base+a+3] = NewFloat(idx)
					}
				} else {
					if idx >= limit {
						if err := vm.CheckInterrupt(); err != nil {
							return nil, err
						}
						frame.pc -= bx + 1
						vm.stack[frame.base+a+3] = NewFloat(idx)
					}
				}
			}

		case compiler.OP_FORPREP:
			a, bx := inst.A(), inst.Bx()
			// R[A] = init, R[A+1] = limit, R[A+2] = step
			init := vm.stack[frame.base+a]
			limit := vm.stack[frame.base+a+1]
			step := vm.stack[frame.base+a+2]

			// Coerce string operands to numbers.
			// Lua 5.4: if init or step was a string, force float mode
			// (string limit alone does not force float).
			initWasString := init.IsString()
			stepWasString := step.IsString()
			if init.IsString() {
				if nv, ok := StringToNumericValue(init.AsString()); ok {
					init = nv
				}
			}
			if limit.IsString() {
				if nv, ok := StringToNumericValue(limit.AsString()); ok {
					limit = nv
				}
			}
			if step.IsString() {
				if nv, ok := StringToNumericValue(step.AsString()); ok {
					step = nv
				}
			}

			// Lua 5.4: if init and step are integer TYPE (not just convertible),
			// use integer mode. Float values like 1.0 use float mode.
			// However, if init or step was originally a string, force float mode
			// even if the string converted to an integer.
			if init.IsInt() && step.IsInt() && !initWasString && !stepWasString {
				initI := init.AsInt()
				stepI := step.AsInt()
				if stepI == 0 {
					return nil, vm.runtimeError("'for' step is zero")
				}
				// Try to convert limit to integer
				limitI, limitIsInt := limit.ToInt()
				if !limitIsInt && limit.IsNumber() {
					// Float limit: convert using floor (step>0) or ceil (step<0)
					limitF := limit.AsFloat()
					if math.IsInf(limitF, 1) {
						// +Inf: step>0 → use MaxInt64; step<0 → skip
						if stepI < 0 {
							frame.pc += bx + 1
							break
						}
						limitI = math.MaxInt64
						limitIsInt = true
					} else if math.IsInf(limitF, -1) {
						// -Inf: step>0 → skip; step<0 → use MinInt64
						if stepI > 0 {
							frame.pc += bx + 1
							break
						}
						limitI = math.MinInt64
						limitIsInt = true
					} else if math.IsNaN(limitF) {
						// NaN limit with integer init/step: Lua 5.4's
						// luaV_forlimit treats NaN as "smaller than min int"
						// (since 0 < NaN is false). So limit = MinInt64.
						// For positive step: stopnow = true → skip loop.
						// For negative step: stopnow = false → enter loop
						// with limit = MinInt64 (always >= comparison true).
						if stepI >= 0 {
							// Positive step: skip loop
							vm.stack[frame.base+a] = NewInt(initI)
							vm.stack[frame.base+a+1] = NewInt(math.MinInt64)
							vm.stack[frame.base+a+2] = NewInt(stepI)
							frame.pc += bx + 1
							break
						}
						// Negative step: enter loop with limit = MinInt64
						limitI = math.MinInt64
						limitIsInt = true
					} else if stepI > 0 {
						fl := math.Floor(limitF)
						if fl < float64(math.MinInt64) {
							// Limit too negative, loop never runs
							frame.pc += bx + 1
							break
						} else if fl >= float64(math.MaxInt64) {
							// >= not >: float64(math.MaxInt64) rounds up to 2^63,
							// so fl == 2^63 must clamp here. Otherwise int64(fl)
							// below overflows to MinInt64 and skips the loop.
							limitI = math.MaxInt64
						} else {
							limitI = int64(fl)
						}
						limitIsInt = true
					} else {
						cl := math.Ceil(limitF)
						if cl >= float64(math.MaxInt64) {
							// >= not >: float64(math.MaxInt64) rounds up to 2^63,
							// so cl == 2^63 must take this branch — with a
							// negative step a limit of +2^63 means the loop
							// never runs.
							frame.pc += bx + 1
							break
						} else if cl < float64(math.MinInt64) {
							limitI = math.MinInt64
						} else {
							limitI = int64(cl)
						}
						limitIsInt = true
					}
				} else if !limitIsInt {
					return nil, vm.runtimeError("bad 'for' limit (number expected, got %s)", vm.ObjTypeName(limit))
				}

				// Integer for loop: counter-based (Lua 5.4 semantics)
				// R[A] = current index
				// R[A+1] = remaining iterations counter = (limit - init) / step
				// R[A+2] = step
				// R[A+3] = visible i (copy of index)
				shouldSkip := false
				if stepI >= 0 {
					if initI > limitI {
						shouldSkip = true
					}
				} else {
					if initI < limitI {
						shouldSkip = true
					}
				}
				if shouldSkip {
					vm.stack[frame.base+a] = NewInt(initI)
					vm.stack[frame.base+a+1] = NewInt(0)
					vm.stack[frame.base+a+2] = NewInt(stepI)
					frame.pc += bx + 1
				} else {
					// Compute counter using unsigned division to avoid overflow.
					// Lua 5.4 uses: (uint64)(limit - init) / (uint64)(step)
					var counter int64
					if stepI > 0 {
						counter = int64(uint64(limitI-initI) / uint64(stepI))
					} else {
						counter = int64(uint64(initI-limitI) / uint64(-stepI))
					}
					vm.stack[frame.base+a] = NewInt(initI)
					vm.stack[frame.base+a+1] = NewInt(counter)
					vm.stack[frame.base+a+2] = NewInt(stepI)
					vm.stack[frame.base+a+3] = NewInt(initI)
				}
			} else {
				// Float for loop — validate in Lua 5.4 order: limit, step, initial
				limitF, ok2 := limit.ToNumber()
				stepF, ok3 := step.ToNumber()
				initF, ok1 := init.ToNumber()
				if !ok2 {
					return nil, vm.runtimeError("bad 'for' limit (number expected, got %s)", vm.ObjTypeName(limit))
				}
				if !ok3 {
					return nil, vm.runtimeError("bad 'for' step (number expected, got %s)", vm.ObjTypeName(step))
				}
				if !ok1 {
					return nil, vm.runtimeError("bad 'for' initial value (number expected, got %s)", vm.ObjTypeName(init))
				}
				if stepF == 0 {
					return nil, vm.runtimeError("'for' step is zero")
				}
				vm.stack[frame.base+a] = NewFloat(initF)
				vm.stack[frame.base+a+1] = NewFloat(limitF)
				vm.stack[frame.base+a+2] = NewFloat(stepF)
				// Use C-style float comparison semantics: NaN comparisons
				// return false, so "should we skip?" checks fail and the
				// loop enters. This matches Lua 5.4's luai_numlt behavior.
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
					base:      nativeBase,
					argc:      2, // iterator always called with (state, ctrl)
					funcValue: fn,
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
			} else if fn.IsTable() {
				// Check for __call metamethod
				mt := fn.AsTable().Metatable()
				if mt != nil {
					callMM := mt.Get(metaCall)
					if !callMM.IsNil() {
						// Call __call(self, state, ctrl)
						results, err = vm.ProtectedCall(callMM, []Value{fn, state, ctrl})
						if err != nil {
							return nil, err
						}
					} else {
						return nil, vm.runtimeError("attempt to call a %s value (for iterator 'for iterator')", vm.ObjTypeName(fn))
					}
				} else {
					return nil, vm.runtimeError("attempt to call a %s value (for iterator 'for iterator')", vm.ObjTypeName(fn))
				}
			} else {
				return nil, vm.runtimeError("attempt to call a %s value (for iterator 'for iterator')", vm.ObjTypeName(fn))
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
				if err := vm.CheckInterrupt(); err != nil {
					return nil, err
				}
				vm.stack[frame.base+a+2] = vm.stack[frame.base+a+4]
				frame.pc -= bx + 1
			}

		case compiler.OP_SETLIST:
			a := inst.A()
			// IvABC format: extract vB (count) and vC (offset)
			vB := inst.VB()
			vC := inst.VC()
			k := inst.K()

			tbl := vm.stack[frame.base+a].AsTable()
			if tbl == nil {
				return nil, vm.runtimeError("attempt to index a non-table value")
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
			// Fast path: OP_SETLIST always follows OP_NEWTABLE, so table is always *Table.
			// Write directly into the array part including nils so that table
			// constructors like {nil, "a"} produce a correctly-sized array.
			if ct, ok := tbl.(*Table); ok {
				needed := offset + n - 1 // last 1-based index
				if needed > len(ct.array) {
					newArr := make([]Value, needed)
					copy(newArr, ct.array)
					ct.array = newArr
				}
				for i := 0; i < n; i++ {
					idx := offset + i
					ct.array[idx-1] = vm.stack[frame.base+a+1+i]
					// OP_SETLIST writes array slots directly; clear any stale integer
					// hash entry for the same index (from earlier keyed fields in the
					// same constructor) so later shrink/lookup cannot expose old values.
					ct.setIntHash(int64(idx), Nil)
				}
				// Do not rehash hash keys into array here. In mixed constructors
				// like {"a", false, [3] = {}, [4] = {}}, Lua 5.4 keeps explicit
				// keyed entries from the hash part from influencing the constructor's
				// list border selection.
			} else {
				for i := 0; i < n; i++ {
					if err := tbl.Set(NewInt(int64(offset+i)), vm.stack[frame.base+a+1+i]); err != nil {
						return nil, err
					}
				}
			}

		case compiler.OP_CLOSURE:
			a, bx := inst.A(), inst.Bx()
			vm.stack[frame.base+a] = NewFunction(vm.makeClosure(frame, proto.Protos[bx]))

		case compiler.OP_VARARG:
			a, c := inst.A(), inst.C()
			// Copy varargs to R[A], ..., R[A+C-2]
			// If C=0, copy all varargs and set top

			numWanted := c - 1
			if c == 0 {
				numWanted = frame.numVararg
				vm.top = frame.base + a + numWanted
			}

			// Ensure the stack can hold all vararg values.
			if needed := frame.base + a + numWanted; needed > len(vm.stack) {
				vm.ensureStack(needed)
			}

			for i := 0; i < numWanted; i++ {
				if i < frame.numVararg {
					vm.stack[frame.base+a+i] = frame.varargs[i]
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

// ensureStack grows the VM stack to hold at least n slots.
// Panics on stack overflow when limits are set. This panic is always caught
// by ProtectedCall's recover() — ensureStack is never called outside a
// protected boundary. The panic is the correct mechanism because it mirrors
// Lua's longjmp-based error propagation and avoids threading error returns
// through the entire instruction dispatch loop.
func (vm *VM) ensureStack(n int) {
	limit := vm.limits.MaxStackSlots
	if limit == 0 {
		limit = DefaultMaxStackSlots
	}
	if limit > 0 && n >= limit {
		panic(fmt.Sprintf("C stack overflow: %d slots exceeds limit %d",
			n, limit))
	}
	for len(vm.stack) <= n {
		vm.stack = append(vm.stack, make([]Value, stackGrowChunk)...)
	}
}

// EnsureStack grows the stack so that index n is valid.
// Must be called from within a ProtectedCall boundary (i.e., from a
// NativeFunc). Panics on stack overflow, caught by ProtectedCall's recover.
func (vm *VM) EnsureStack(n int) {
	vm.ensureStack(n)
}

// CheckStack returns true if the stack can accommodate n slots without
// exceeding the configured limit. Does not modify the stack.
func (vm *VM) CheckStack(n int) bool {
	limit := vm.limits.MaxStackSlots
	if limit == 0 {
		limit = DefaultMaxStackSlots
	}
	return limit < 0 || n < limit
}

// doCall dispatches an OP_CALL instruction. It collects arguments from the
// stack, calls the target (closure, native, or __call metamethod), and stores
// the results back into the caller's registers. It returns only an error: the
// results are written directly into registers, so returning the result slice
// would force the stack-local result buffer to escape to the heap.
func (vm *VM) doCall(frame *callFrame, a, b, c int) error {
	fn := vm.stack[frame.base+a]

	// Collect arguments using a stack-allocated buffer for the common case
	// (≤8 args) to avoid a heap allocation per function call.
	var argBuf [8]Value
	var args []Value
	var nArgs int
	if b == 0 {
		nArgs = vm.top - (frame.base + a + 1)
	} else {
		nArgs = b - 1
	}
	if nArgs <= len(argBuf) {
		args = argBuf[:nArgs]
	} else {
		args = make([]Value, nArgs)
	}
	if b == 0 {
		copy(args, vm.stack[frame.base+a+1:vm.top])
	} else {
		copy(args, vm.stack[frame.base+a+1:frame.base+a+b])
	}

	// Restore vm.top to the calling frame's proper level. A previous CALL with
	// c=0 (variable results) may have lowered vm.top to indicate the result count.
	// We must restore it AFTER collecting b=0 args (which reads vm.top) but BEFORE
	// dispatching any calls, so that vm.call gets the correct base position and
	// native functions that call ProtectedCall won't overlap with caller registers.
	frameTop := frame.base + frame.closure.Proto.MaxStack
	vm.top = frameTop

	var results []Value
	var err error

dispatch:
	if fn.IsFunction() {
		// vm.top is at frameTop, so vm.call will place the new frame right after
		// the calling frame's full register space. Args are copied into a slice
		// so they're safe regardless of where the new frame starts.
		results, err = vm.call(fn.AsClosure(), args, c-1)
	} else if fn.IsNativeFunc() {
		// Set up for native function
		// Push a temporary call frame so vm.Base() works correctly for native functions.
		// We keep vm.top at its current value (past the calling Lua frame's registers)
		// so that if the native function calls back into Lua via ProtectedCall,
		// the new Lua frame won't overlap with the caller's register space.
		// ArgCount uses the stored argc instead of computing from vm.top.
		nativeBase := frame.base + a
		nativeFrame := callFrame{
			base:      nativeBase,
			argc:      len(args),
			funcValue: fn,
			ftransfer: 1,         // args start at getlocal index 1 (base+1 = first arg)
			ntransfer: len(args), // number of arguments
		}
		vm.callStack = append(vm.callStack, nativeFrame)

		// Clear slots beyond the arguments to prevent stale data from affecting
		// optional argument checks (e.g., if !v.Get(3).IsNil())
		clearStart := nativeBase + 1 + len(args)
		clearEnd := clearStart + 4
		if clearEnd > len(vm.stack) {
			clearEnd = len(vm.stack)
		}
		for i := clearStart; i < clearEnd; i++ {
			vm.stack[i] = Nil
		}

		// Constrain vm.top for the native frame so getlocal only sees
		// actual arguments (matching Lua 5.4's L->top management).
		savedTop := vm.top
		vm.top = nativeBase + 1 + len(args)

		// Fire call hook for native function (after frame is pushed).
		// The call hook may mutate arguments via debug.setlocal.
		vm.fireCallHook()

		// Save arguments only if hooks are active — the savedArgs copy is
		// only needed when a return hook will fire (so it can see [args,
		// results] on the stack). When no hooks are active at entry, the
		// call hook didn't fire (gated by hookMask), so the `args` slice
		// captured before the call is still the authoritative pre-call
		// state and can be reused if the native enables hooks mid-call.
		hooksActiveBefore := vm.hookMask != 0
		var savedArgs []Value
		if hooksActiveBefore {
			savedArgs = make([]Value, len(args))
			copy(savedArgs, vm.stack[nativeBase+1:nativeBase+1+len(args)])
		}

		nResults := fn.AsNativeFunc()(vm)

		// Re-check hookMask: a native (e.g. debug.sethook) can enable hooks
		// mid-call, in which case the return hook must still fire.
		if vm.hookMask == 0 {
			// Fast path — no debug hooks. The native function wrote its
			// nResults return values into vm.stack[nativeBase..nativeBase+
			// nResults], and the caller's result registers begin at
			// nativeBase (== frame.base+a), so the results are already in
			// their final position: no buffer copy or relocation needed.
			// This keeps the result buffer off the heap entirely (the hot
			// path for any native-call-heavy Lua workload).
			vm.top = savedTop
			vm.callStack = vm.callStack[:len(vm.callStack)-1]

			nWanted := c - 1
			if c == 0 {
				nWanted = nResults
				vm.top = frame.base + a + nResults
			}
			if needed := frame.base + a + nWanted; needed > len(vm.stack) {
				vm.ensureStack(needed)
			}
			// Pad with Nil when the caller wants more results than produced.
			for i := nResults; i < nWanted; i++ {
				vm.stack[frame.base+a+i] = Nil
			}
			// Clear dead registers above the result area so Go's GC does not
			// retain values referenced only by stale temporaries.
			for i := frame.base + a + nWanted; i < frameTop && i < len(vm.stack); i++ {
				vm.stack[i] = Nil
			}
			return nil
		}

		// Slow path — debug hooks are active. Copy the results into a heap
		// buffer so the relocation below (which rewrites the stack so the
		// return hook sees [args..., results...]) does not lose them. The
		// allocation is acceptable here: this path only runs under an active
		// debug hook.
		results = make([]Value, nResults)
		copy(results, vm.stack[nativeBase:nativeBase+nResults])

		nf := &vm.callStack[len(vm.callStack)-1]
		if savedArgs == nil {
			// No hooks at entry → fireCallHook did nothing → args slice
			// still matches the pre-call stack state.
			savedArgs = args
		}
		// Restore arguments and place return values after them so the
		// return hook sees [args..., results...] via getlocal, matching
		// Lua 5.4 semantics.
		copy(vm.stack[nativeBase+1:nativeBase+1+nf.argc], savedArgs)
		retStart := nativeBase + 1 + nf.argc
		retEnd := retStart + nResults
		if retEnd > len(vm.stack) {
			retEnd = len(vm.stack)
		}
		for i := 0; i < retEnd-retStart; i++ {
			vm.stack[retStart+i] = results[i]
		}
		vm.top = retEnd
		if nResults > 0 {
			nf.ftransfer = 1 + nf.argc
			nf.ntransfer = nResults
		} else {
			nf.ftransfer = 0
			nf.ntransfer = 0
		}

		// Fire return hook for native function before popping its frame
		vm.fireReturnHook()
		vm.top = savedTop

		// Pop the native frame
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
	} else {
		// Check for __call metamethod
		mm := vm.getMetafield(fn, MetaCall)
		if mm.IsNil() {
			return vm.runtimeError("attempt to call a %s value%s", vm.ObjTypeName(fn), vm.varInfo(a))
		}

		// Build new args: prepend fn (self) so the call becomes mm(fn, args...)
		newArgs := make([]Value, len(args)+1)
		newArgs[0] = fn
		copy(newArgs[1:], args)

		// Recursively dispatch mm with newArgs so that __call chains of
		// arbitrary depth are resolved (mm may itself be a table with __call).
		// Write mm + newArgs into the stack so the callee sees them via Get.
		vm.ensureStack(frame.base + a + 1 + len(newArgs) + 4)
		vm.stack[frame.base+a] = mm
		for i, arg := range newArgs {
			vm.stack[frame.base+a+1+i] = arg
		}
		fn = mm
		args = newArgs
		goto dispatch
	}

	if err != nil {
		return err
	}

	// Store results
	nWanted := c - 1
	if c == 0 {
		// Variable results - set top
		nWanted = len(results)
		vm.top = frame.base + a + nWanted
	}

	// Ensure the stack can hold all result slots before writing.
	if needed := frame.base + a + nWanted; needed > len(vm.stack) {
		vm.ensureStack(needed)
	}

	for i := 0; i < nWanted; i++ {
		if i < len(results) {
			vm.stack[frame.base+a+i] = results[i]
		} else {
			vm.stack[frame.base+a+i] = Nil
		}
	}

	// Clear dead registers above the result area up to the caller frame's
	// max stack. Function calls leave arguments and temporaries in registers
	// that Go's GC still traces (unlike C Lua where the GC only traces up
	// to L->top). Without this, weak table values and __gc tables referenced
	// only by dead temporaries are never collected.
	for i := frame.base + a + nWanted; i < frameTop && i < len(vm.stack); i++ {
		vm.stack[i] = Nil
	}

	return nil
}

// localName returns the name of the local variable occupying register reg
// at the given pc, or "?" if not found. Mirrors Lua 5.4's luaF_getlocalname.
func localName(proto *compiler.Proto, reg int, pc int) string {
	n := reg + 1 // 1-based local number
	for i := 0; i < len(proto.Locals) && proto.Locals[i].StartPC <= pc; i++ {
		if pc < proto.Locals[i].EndPC { // active at this PC
			n--
			if n == 0 {
				return proto.Locals[i].Name
			}
		}
	}
	return "?"
}
