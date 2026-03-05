package vm

import (
	"fmt"
	"math"
	"strings"

	"github.com/iceisfun/golua/compiler"
)

// call invokes a closure with the given arguments and returns results.
func (vm *VM) call(closure *Closure, args []Value, nResults int) ([]Value, error) {
	vm.checkCallDepth()

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
			varargPos = base + proto.MaxStack + VarargBufferOffset
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
		closure:      closure,
		pc:           0,
		base:         base,
		nResults:     nResults,
		isVararg:     proto.IsVarArg,
		varargPos:    varargPos,
		numVararg:    numVararg,
		argc:         UseVMTop, // Lua frames use vm.top for ArgCount
		callName:     vm.pendingCallName,
		callNameWhat: vm.pendingCallNameWhat,
	}
	vm.pendingCallName = ""
	vm.pendingCallNameWhat = ""
	vm.callStack = append(vm.callStack, frame)

	// Update vm.top to point past this frame's registers
	// This ensures nested calls get non-overlapping stack regions
	vm.top = base + proto.MaxStack

	// Fire call hook after frame is pushed
	vm.fireCallHook()

	// Execute
	results, err := vm.execute()

	// Pop call frame and restore top
	vm.callStack = vm.callStack[:len(vm.callStack)-1]
	vm.top = savedTop

	// Clear dead stack slots so Go's GC can collect objects
	// that were only reachable from the called function's frame.
	clearEnd := base + proto.MaxStack
	if clearEnd > len(vm.stack) {
		clearEnd = len(vm.stack)
	}
	for i := vm.top; i < clearEnd; i++ {
		vm.stack[i] = Value{}
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
	return nil
}

// execute runs the current call frame until it returns.
func (vm *VM) execute() ([]Value, error) {
	for {
		frame := &vm.callStack[len(vm.callStack)-1]
		proto := frame.closure.Proto
		code := proto.Code
		consts := frame.closure.ConstValues()

		if frame.pc >= len(code) {
			return nil, nil
		}

		inst := code[frame.pc]
		frame.pc++

		// Hook dispatch: line and count hooks (fast path: hookMask == 0 skips entirely)
		if vm.hookMask != 0 && !vm.inHook {
			if vm.checkLineCountHooks(proto, frame.pc-1) {
				// Re-fetch after hook callback may have modified call stack
				frame = &vm.callStack[len(vm.callStack)-1]
				proto = frame.closure.Proto
				code = proto.Code
				consts = frame.closure.ConstValues()
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
			} else {
				return nil, vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(table))
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
			} else if mm := vm.getMetafield(table, "__index"); !mm.IsNil() {
				// Type metatable __index
				val, err := vm.resolveIndex(mm, table, key)
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
				proto = frame.closure.Proto
				code = proto.Code
				consts = frame.closure.ConstValues()
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
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
			} else if mm := vm.getMetafield(table, "__index"); !mm.IsNil() {
				val, err := vm.resolveIndex(mm, table, NewInt(int64(c)))
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
				proto = frame.closure.Proto
				code = proto.Code
				consts = frame.closure.ConstValues()
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
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
			} else if mm := vm.getMetafield(table, "__index"); !mm.IsNil() {
				val, err := vm.resolveIndex(mm, table, NewString(key))
				if err != nil {
					return nil, err
				}
				vm.stack[frame.base+a] = val
				frame = &vm.callStack[len(vm.callStack)-1]
				proto = frame.closure.Proto
				code = proto.Code
				consts = frame.closure.ConstValues()
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
			}

		case compiler.OP_SETTABUP:
			a, b, c := inst.A(), inst.B(), inst.C()
			table := frame.closure.Upvalues[a].Get()
			key := consts[b]
			value := vm.getRK(frame, c, inst.K())
			if t := table.AsTable(); t != nil {
				if err := vm.tableSet(t, key, value); err != nil {
					return nil, err
				}
			} else {
				return nil, vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(table))
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
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(a))
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
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(a))
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
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(a))
			}

		case compiler.OP_NEWTABLE:
			a := inst.A()
			// vB and vC encode size hints (we ignore them for now)
			vm.stack[frame.base+a] = NewTable(NewEmptyTable())

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
				// String method call - use string metatable
				vm.stack[frame.base+a] = vm.stringMeta.Get(NewString(key))
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
			} else {
				return nil, vm.runtimeError("attempt to index a %s value%s", vm.ObjTypeName(table), vm.varInfo(b))
			}

		case compiler.OP_ADDI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			origV := vm.stack[frame.base+b]
			v := origV
			// Coerce string operands to numbers (preserving int/float type),
			// matching the string coercion in arith() for OP_ADD.
			if v.IsString() {
				if nv, ok := StringToNumericValue(v.AsString()); ok {
					v = nv
				}
			}
			if v.IsInt() {
				vm.stack[frame.base+a] = NewInt(v.AsInt() + int64(sc))
			} else if n, ok := v.ToNumber(); ok {
				vm.stack[frame.base+a] = NewFloat(n + float64(sc))
			} else {
				// Try __add metamethod with correct operand order.
				// The next instruction is OP_MMBINI whose k flag indicates
				// whether the immediate was the left operand (k=1) or right (k=0).
				immVal := NewInt(int64(sc))
				arg1, arg2 := origV, immVal
				if frame.pc < len(code) {
					nextInst := code[frame.pc]
					if nextInst.OpCode() == compiler.OP_MMBINI && nextInst.K() == 1 {
						arg1, arg2 = immVal, origV
					}
				}
				if mm := vm.getArithMetamethod(arg1, arg2, "__add"); !mm.IsNil() {
					if !mm.IsCallable() {
						return nil, vm.runtimeError("attempt to call a %s value (metamethod 'add')", mm.Type())
					}
					result, err := vm.callMetamethod("add", mm, arg1, arg2)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = result
					frame = &vm.callStack[len(vm.callStack)-1]
					proto = frame.closure.Proto
					code = proto.Code
					consts = frame.closure.ConstValues()
				} else {
					return nil, vm.runtimeError("attempt to perform arithmetic on a %s value%s", vm.ObjTypeName(origV), vm.varInfo(b))
				}
			}

		case compiler.OP_ADDK, compiler.OP_SUBK, compiler.OP_MULK, compiler.OP_MODK,
			compiler.OP_POWK, compiler.OP_DIVK, compiler.OP_IDIVK:
			a, b, c := inst.A(), inst.B(), inst.C()
			v := vm.stack[frame.base+b]
			kv := consts[c]
			result, err := vm.arithK(op, v, kv)
			if err != nil {
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_BANDK, compiler.OP_BORK, compiler.OP_BXORK:
			a, b, c := inst.A(), inst.B(), inst.C()
			v := vm.stack[frame.base+b]
			kv := consts[c]
			result, err := vm.bitwiseK(op, v, kv)
			if err != nil {
				// Enhance "number has no integer representation" with register name
				if strings.Contains(err.Error(), "number has no integer representation") {
					if v.IsNumber() {
						if _, ok := v.ToInt(); !ok {
							return nil, vm.runtimeErrorForNumber(b)
						}
					}
				}
				return nil, err
			}
			vm.stack[frame.base+a] = result

		case compiler.OP_SHLI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			// Lua 5.4: bitwise ops do NOT coerce strings
			if !v.IsString() {
				if i, ok := v.ToInt(); ok {
					vm.stack[frame.base+a] = NewInt(int64(sc) << uint(i))
				} else if v.IsNumber() {
					return nil, vm.runtimeErrorForNumber(b)
				} else {
					return nil, vm.runtimeError("attempt to perform bitwise operation on a %s value", vm.ObjTypeName(v))
				}
			} else {
				return nil, vm.runtimeError("attempt to perform bitwise operation on a %s value", vm.ObjTypeName(v))
			}

		case compiler.OP_SHRI:
			a, b := inst.A(), inst.B()
			sc := inst.SC()
			v := vm.stack[frame.base+b]
			// Lua 5.4: bitwise ops do NOT coerce strings
			if !v.IsString() {
				if i, ok := v.ToInt(); ok {
					vm.stack[frame.base+a] = NewInt(int64(uint64(i) >> uint(sc)))
				} else if v.IsNumber() {
					return nil, vm.runtimeErrorForNumber(b)
				} else {
					return nil, vm.runtimeError("attempt to perform bitwise operation on a %s value", vm.ObjTypeName(v))
				}
			} else {
				return nil, vm.runtimeError("attempt to perform bitwise operation on a %s value", vm.ObjTypeName(v))
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
				// Enhance "number has no integer representation" with register name
				if strings.Contains(err.Error(), "number has no integer representation") {
					// Try the first operand, then the second
					if v1.IsNumber() {
						if _, ok := v1.ToInt(); !ok {
							return nil, vm.runtimeErrorForNumber(b)
						}
					}
					if v2.IsNumber() {
						if _, ok := v2.ToInt(); !ok {
							return nil, vm.runtimeErrorForNumber(c)
						}
					}
				}
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
			} else if v.IsString() {
				if i, ok := v.ToInt(); ok {
					vm.stack[frame.base+a] = NewInt(-i)
				} else if n, ok := v.ToNumber(); ok {
					vm.stack[frame.base+a] = NewFloat(-n)
				} else {
					return nil, vm.runtimeError("attempt to perform arithmetic on a %s value", vm.ObjTypeName(v))
				}
			} else if mm := vm.getMetafield(v, "__unm"); !mm.IsNil() {
				if !mm.IsCallable() {
					return nil, vm.runtimeError("attempt to call a %s value (metamethod 'unm')", mm.Type())
				}
				result, err := vm.callMetamethod("unm", mm, v, v)
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
				if mm := vm.getMetafield(v, "__bnot"); !mm.IsNil() {
					if !mm.IsCallable() {
						return nil, vm.runtimeError("attempt to call a %s value (metamethod 'bnot')", mm.Type())
					}
					result, err := vm.callMetamethod("bnot", mm, v, v)
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
				op := "__len"
				mm := vm.getMetafield(v, op)
				if !mm.IsNil() {
					if !mm.IsCallable() {
						return nil, vm.runtimeError("attempt to call a %s value (metamethod 'len')", mm.Type())
					}
					res, err := vm.callMetamethod("len", mm, v, v)
					if err != nil {
						return nil, err
					}
					vm.stack[frame.base+a] = res
				} else if v.IsTable() {
					vm.stack[frame.base+a] = NewInt(int64(v.AsTable().Len()))
				} else {
					return nil, vm.runtimeError("attempt to get length of a %s value%s", v.Type(), vm.varInfo(b))
				}
			}

		case compiler.OP_CONCAT:
			a, b := inst.A(), inst.B()
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
					if totalLen > (1<<30) - l {
						return nil, vm.runtimeError("string length overflow")
					}
					totalLen += l
				} else {
					if totalLen > (1<<30) - 20 {
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
						res, err := vm.concat(prev, current)
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
					if mt == nil || mt.Get(metaClose).IsNil() {
						needErr = true
					}
				} else {
					needErr = true
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
			if !v.IsNumber() {
				return nil, vm.runtimeError("attempt to compare %s with number", v.Type())
			}
			var lt bool
			if v.IsInt() {
				lt = v.AsInt() < int64(sb)
			} else {
				lt = v.AsFloat() < float64(sb)
			}
			if lt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_LEI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			if !v.IsNumber() {
				return nil, vm.runtimeError("attempt to compare %s with number", v.Type())
			}
			var le bool
			if v.IsInt() {
				le = v.AsInt() <= int64(sb)
			} else {
				le = v.AsFloat() <= float64(sb)
			}
			if le != (k == 1) {
				frame.pc++
			}

		case compiler.OP_GTI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			if !v.IsNumber() {
				return nil, vm.runtimeError("attempt to compare %s with number", v.Type())
			}
			var gt bool
			if v.IsInt() {
				gt = v.AsInt() > int64(sb)
			} else {
				gt = v.AsFloat() > float64(sb)
			}
			if gt != (k == 1) {
				frame.pc++
			}

		case compiler.OP_GEI:
			a, k := inst.A(), inst.K()
			sb := inst.SB()
			v := vm.stack[frame.base+a]
			if !v.IsNumber() {
				return nil, vm.runtimeError("attempt to compare %s with number", v.Type())
			}
			var ge bool
			if v.IsInt() {
				ge = v.AsInt() >= int64(sb)
			} else {
				ge = v.AsFloat() >= float64(sb)
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
			results, err := vm.doCall(frame, a, b, c)
			if err != nil {
				return nil, err
			}
			_ = results

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
			// Fire tail call hook before reusing frame
			vm.fireTailCallHook()
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

			// Dispatch loop for __call support
			for {
				if fn.IsFunction() {
					closure := fn.AsClosure()
					// Reuse the frame
					frame.closure = closure
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
					break // Continue outer loop (instruction loop)
				} else if fn.IsNativeFunc() {
					// Native function tail call - can't truly optimize, just call
					nf := fn.AsNativeFunc()
					// Reuse current frame's base for the native call
					vm.stack[frame.base] = fn
					for i, arg := range args {
						vm.stack[frame.base+1+i] = arg
					}
					frame.argc = len(args)

					// Clear slots beyond the arguments to prevent stale register
					// data from being seen as optional arguments by native functions
					// (e.g., table.concat checking v.Get(3) for optional start index)
					clearStart := frame.base + 1 + len(args)
					clearEnd := clearStart + 4
					if clearEnd > len(vm.stack) {
						clearEnd = len(vm.stack)
					}
					for i := clearStart; i < clearEnd; i++ {
						vm.stack[i] = Nil
					}

					// The current frame already exists, just call the native function
					// vm.Base() will correctly return frame.base
					nResults := nf(vm)
					results := make([]Value, nResults)
					copy(results, vm.stack[frame.base:frame.base+nResults])
					return results, nil
				} else {
					// Check for __call metamethod
					op := "__call"
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
					return nil, vm.runtimeError("attempt to call a %s value%s", fn.Type(), vi)
				}
			}
			if fn.IsFunction() {
				// We broke out of the inner loop to continue the outer instruction loop
				// because IsFunction setup sets frame.pc=0.
				continue
			}

		case compiler.OP_RETURN:
			a, b, c := inst.A(), inst.B(), inst.C()
			_ = c // c contains info about closing upvalues

			// Collect return values BEFORE closing upvalues, since closeUpvalues
			// may modify the stack (running __close metamethods).
			var results []Value
			if b == 0 {
				// Return values from a to top
				results = make([]Value, vm.top-(frame.base+a))
				copy(results, vm.stack[frame.base+a:vm.top])
			} else {
				results = make([]Value, b-1)
				copy(results, vm.stack[frame.base+a:frame.base+a+b-1])
			}

			// Close upvalues and run __close metamethods BEFORE the return hook.
			// Lua 5.4 runs __close before the return hook for the function itself.
			vm.closeUpvalues(frame.base)

			// Fire return hook after __close metamethods
			vm.fireReturnHook()

			return results, nil

		case compiler.OP_RETURN0:
			vm.closeUpvalues(frame.base)
			vm.fireReturnHook()
			return nil, nil

		case compiler.OP_RETURN1:
			a := inst.A()
			result := vm.stack[frame.base+a]
			vm.closeUpvalues(frame.base)
			vm.fireReturnHook()
			return []Value{result}, nil

		case compiler.OP_FORLOOP:
			a, bx := inst.A(), inst.Bx()
			// R[A] = index, R[A+1] = limit, R[A+2] = step
			stepVal := vm.stack[frame.base+a+2]
			if stepVal.IsInt() {
				// Integer for loop
				idx := vm.stack[frame.base+a].AsInt()
				limit := vm.stack[frame.base+a+1].AsInt()
				step := stepVal.AsInt()
				// Check for overflow before adding step
				if step > 0 && idx > math.MaxInt64-step {
					// Would overflow — exit loop
				} else if step < 0 && idx < math.MinInt64-step {
					// Would underflow — exit loop
				} else {
					idx += step
					vm.stack[frame.base+a] = NewInt(idx)
					if step >= 0 {
						if idx <= limit {
							if err := vm.CheckInterrupt(); err != nil {
								return nil, err
							}
							frame.pc -= bx + 1
							vm.stack[frame.base+a+3] = NewInt(idx)
						}
					} else {
						if idx >= limit {
							if err := vm.CheckInterrupt(); err != nil {
								return nil, err
							}
							frame.pc -= bx + 1
							vm.stack[frame.base+a+3] = NewInt(idx)
						}
					}
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

			// Coerce string operands to numbers
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
			if init.IsInt() && step.IsInt() {
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
						return nil, vm.runtimeError("bad 'for' limit (number expected, got %s)", limit.Type())
					} else if stepI > 0 {
						fl := math.Floor(limitF)
						if fl < float64(math.MinInt64) {
							// Limit too negative, loop never runs
							frame.pc += bx + 1
							break
						} else if fl > float64(math.MaxInt64) {
							limitI = math.MaxInt64
						} else {
							limitI = int64(fl)
						}
						limitIsInt = true
					} else {
						cl := math.Ceil(limitF)
						if cl > float64(math.MaxInt64) {
							// Limit too positive, loop never runs
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
					return nil, vm.runtimeError("bad 'for' limit (number expected, got %s)", limit.Type())
				}

				// Integer for loop
				vm.stack[frame.base+a] = NewInt(initI)
				vm.stack[frame.base+a+1] = NewInt(limitI)
				vm.stack[frame.base+a+2] = NewInt(stepI)
				if stepI >= 0 {
					if initI > limitI {
						frame.pc += bx + 1
					} else {
						vm.stack[frame.base+a+3] = NewInt(initI)
					}
				} else {
					if initI < limitI {
						frame.pc += bx + 1
					} else {
						vm.stack[frame.base+a+3] = NewInt(initI)
					}
				}
			} else {
				// Float for loop
				initF, ok1 := init.ToNumber()
				limitF, ok2 := limit.ToNumber()
				stepF, ok3 := step.ToNumber()
				if !ok1 {
					return nil, vm.runtimeError("bad 'for' initial value (number expected, got %s)", init.Type())
				}
				if !ok2 {
					return nil, vm.runtimeError("bad 'for' limit (number expected, got %s)", limit.Type())
				}
				if !ok3 {
					return nil, vm.runtimeError("bad 'for' step (number expected, got %s)", step.Type())
				}
				if stepF == 0 {
					return nil, vm.runtimeError("'for' step is zero")
				}
				vm.stack[frame.base+a] = NewFloat(initF)
				vm.stack[frame.base+a+1] = NewFloat(limitF)
				vm.stack[frame.base+a+2] = NewFloat(stepF)
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
					base: nativeBase,
					argc: 2, // iterator always called with (state, ctrl)
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
						return nil, vm.runtimeError("attempt to call a %s value", fn.Type())
					}
				} else {
					return nil, vm.runtimeError("attempt to call a %s value", fn.Type())
				}
			} else {
				return nil, vm.runtimeError("attempt to call a %s value", fn.Type())
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
			vB := int((uint32(inst) >> 16) & 0x3F)
			vC := int((uint32(inst) >> 22) & 0x3FF)
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
					ct.array[offset-1+i] = vm.stack[frame.base+a+1+i]
				}
				// Move any consecutive hash entries that now belong in array.
				ct.rehashToArray()
			} else {
				for i := 0; i < n; i++ {
					if err := tbl.Set(NewInt(int64(offset+i)), vm.stack[frame.base+a+1+i]); err != nil {
						return nil, err
					}
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
		panic(fmt.Sprintf("stack overflow: %d slots exceeds limit %d",
			n, limit))
	}
	for len(vm.stack) <= n {
		vm.stack = append(vm.stack, make([]Value, 256)...)
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


// getRK returns either a constant (when k != 0) or a register value.
// Used by instructions that encode an operand as "register or constant".
func (vm *VM) getRK(frame *callFrame, c, k int) Value {
	if k != 0 {
		return frame.closure.ConstValues()[c]
	}
	return vm.stack[frame.base+c]
}
// doCall dispatches an OP_CALL instruction. It collects arguments from the
// stack, calls the target (closure, native, or __call metamethod), and stores
// the results back into the caller's registers.
func (vm *VM) doCall(frame *callFrame, a, b, c int) ([]Value, error) {
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
			ftransfer: 2,         // args start at getlocal index 2 (index 1 = function value)
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

		// Fire call hook for native function (after frame is pushed)
		vm.fireCallHook()

		nResults := fn.AsNativeFunc()(vm)
		results = make([]Value, nResults)
		copy(results, vm.stack[nativeBase:nativeBase+nResults])

		// Set return transfer info: results start at getlocal index 1 (base+0)
		vm.callStack[len(vm.callStack)-1].ftransfer = 1
		vm.callStack[len(vm.callStack)-1].ntransfer = nResults

		// Fire return hook for native function before popping its frame
		vm.fireReturnHook()

		// Pop the native frame
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
	} else {
		// Check for __call metamethod
		mm := vm.getMetafield(fn, "__call")
		if mm.IsNil() {
			return nil, vm.runtimeError("attempt to call a %s value%s", fn.Type(), vm.varInfo(a))
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
