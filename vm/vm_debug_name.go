package vm

import "github.com/iceisfun/golua/compiler"

// This file holds the bytecode name-resolution machinery used by debug
// introspection and tracebacks: inferring the name of a called function from
// the caller's instructions, and describing the value held in a register or
// constant slot. Split out of vm_debug.go for readability.

// funcNameFromCall inspects the caller frame's bytecode to infer the name
// of the function being called. It looks at the instruction at the caller's
// current PC (which should be a CALL or TAILCALL) and traces back to find
// what loaded the function register.
func (vm *VM) funcNameFromCall(callerFrame *callFrame) (name, nameWhat string) {
	if callerFrame.closure == nil {
		return "", ""
	}
	proto := callerFrame.closure.Proto

	// The caller's pc points to the next instruction to execute.
	// The CALL/TAILCALL instruction that triggered the call is at pc-1.
	pc := callerFrame.pc - 1
	if pc < 0 || pc >= len(proto.Code) {
		return "", ""
	}

	inst := proto.Code[pc]
	op := inst.OpCode()

	// For-in iterator call (OP_TFORCALL): name is "for iterator"
	if op == compiler.OP_TFORCALL {
		return "for iterator", "for iterator"
	}

	// Metamethod calls: detect arithmetic/comparison/index metamethods.
	// In GoLua, metamethods are called from within the arith/index handler,
	// so pc-1 points to the triggering instruction. For arithmetic ops, the
	// next instruction (at pc+1) is OP_MMBIN/MMBINI/MMBINK with the tag.
	if mmName := vm.metamethodNameFromOp(proto, pc); mmName != "" {
		return mmName, "metamethod"
	}

	if op != compiler.OP_CALL && op != compiler.OP_TAILCALL {
		return "", ""
	}

	reg := inst.A() // register holding the function

	// Scan backwards for the instruction that wrote to register `reg`.
	for i := pc - 1; i >= 0; i-- {
		prev := proto.Code[i]
		prevOp := prev.OpCode()

		// Skip opcodes whose A operand is not a destination register —
		// jumps, comparisons, the SET* family, returns, etc. Their A bits
		// encode something else (a jump offset, a source operand) and can
		// spuriously alias `reg`. In particular a JMP emitted for a
		// comparison-expression argument sits between the callee load and
		// the CALL; without this skip its offset bits could be mistaken
		// for the instruction that defined the callee, losing the name.
		switch prevOp {
		case compiler.OP_JMP,
			compiler.OP_EQ, compiler.OP_LT, compiler.OP_LE,
			compiler.OP_EQK, compiler.OP_EQI, compiler.OP_LTI,
			compiler.OP_LEI, compiler.OP_GTI, compiler.OP_GEI,
			compiler.OP_TEST,
			compiler.OP_SETTABUP, compiler.OP_SETTABLE,
			compiler.OP_SETI, compiler.OP_SETFIELD,
			compiler.OP_SETLIST, compiler.OP_SETUPVAL,
			compiler.OP_RETURN, compiler.OP_RETURN0, compiler.OP_RETURN1,
			compiler.OP_CLOSE, compiler.OP_TBC,
			compiler.OP_MMBIN, compiler.OP_MMBINI, compiler.OP_MMBINK,
			compiler.OP_TFORCALL,
			compiler.OP_VARARGPREP, compiler.OP_EXTRAARG:
			continue
		}

		prevA := prev.A()

		// Skip instructions that don't write to our target register.
		if prevA != reg {
			// Some opcodes write to A+1 (e.g., SELF writes R[A+1] = R[B]).
			// We only care about the function register (A of the CALL).
			continue
		}

		// If a forward jump can bypass this instruction and still reach the
		// CALL, the name is path-dependent (e.g., from or/and expressions).
		// Suppress annotation in that case, matching Lua 5.4's filterpc.
		if hasForwardBypass(proto, i, pc) {
			return "", ""
		}

		switch prevOp {
		case compiler.OP_GETTABUP:
			// R[A] := UpValue[B][K[C]:string]
			c := prev.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				if isUpvalIdx_ENV(proto, prev.B()) {
					return proto.Constants[c].SVal, "global"
				}
				return proto.Constants[c].SVal, "field"
			}
			return "", ""
		case compiler.OP_GETI:
			return "integer index", "field"
		case compiler.OP_GETTABLE:
			// R[A] = R[B][R[C]] — resolve key name from register C.
			b := prev.B()
			c := prev.C()
			kn := kName(proto, i, c)
			// Check if table is _ENV (global access with high constant index).
			if localName(proto, b, i) == "_ENV" || isUpvalEnv(proto, i, b) {
				if kn != "" {
					return kn, "global"
				}
				return "?", "global"
			}
			// Detect SELF fallback pattern: MOVE base+1,obj + LOADK + GETTABLE base,obj,key.
			// If the previous instruction wrote obj to base+1 (self), this is a method call.
			if kn != "" && isSelfFallback(proto, i, reg, b) {
				return kn, "method"
			}
			if kn != "" {
				return kn, "field"
			}
			return "?", "field"
		case compiler.OP_GETFIELD:
			// R[A] := R[B][K[C]:string]
			c := prev.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				what := "field"
				if localName(proto, prev.B(), i) == "_ENV" || isUpvalEnv(proto, i, prev.B()) {
					what = "global"
				}
				return proto.Constants[c].SVal, what
			}
			return "", ""
		case compiler.OP_SELF:
			// R[A+1] := R[B]; R[A] := R[B][K[C]:string]
			c := prev.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				return proto.Constants[c].SVal, "method"
			}
			return "", ""
		case compiler.OP_MOVE:
			// R[A] := R[B] — try to get the local variable name
			name := localName(proto, prev.B(), i)
			if name != "?" {
				return name, "local"
			}
			return "", ""
		case compiler.OP_GETUPVAL:
			// R[A] := UpValue[B]
			idx := prev.B()
			if idx < len(proto.Upvalues) {
				return proto.Upvalues[idx].Name, "upvalue"
			}
			return "", ""
		default:
			// Found an instruction writing to reg, but can't infer a name.
			return "", ""
		}
	}

	return "", ""
}

// metamethodNameFromOp checks whether the instruction at pc in proto is an
// opcode that invokes a metamethod (arithmetic, comparison, index, unary, etc.).
// Returns the metamethod name without the "__" prefix, or "" if not a metamethod call.
func (vm *VM) metamethodNameFromOp(proto *compiler.Proto, pc int) string {
	if pc < 0 || pc >= len(proto.Code) {
		return ""
	}
	inst := proto.Code[pc]
	op := inst.OpCode()

	switch op {
	// Arithmetic/bitwise ops: the next instruction should be OP_MMBIN/MMBINI/MMBINK
	// containing the metamethod tag in its C field.
	case compiler.OP_ADD, compiler.OP_SUB, compiler.OP_MUL, compiler.OP_MOD,
		compiler.OP_POW, compiler.OP_DIV, compiler.OP_IDIV,
		compiler.OP_BAND, compiler.OP_BOR, compiler.OP_BXOR,
		compiler.OP_SHL, compiler.OP_SHR:
		// Look at next instruction for MMBIN with tag
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			nextOp := next.OpCode()
			if nextOp == compiler.OP_MMBIN || nextOp == compiler.OP_MMBINI || nextOp == compiler.OP_MMBINK {
				tag := decodeBytecodeMetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
				return name
			}
		}
		return ""

	case compiler.OP_ADDI:
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			if next.OpCode() == compiler.OP_MMBINI {
				tag := decodeBytecodeMetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
			}
		}
		return ""

	case compiler.OP_ADDK, compiler.OP_SUBK, compiler.OP_MULK, compiler.OP_MODK,
		compiler.OP_POWK, compiler.OP_DIVK, compiler.OP_IDIVK,
		compiler.OP_BANDK, compiler.OP_BORK, compiler.OP_BXORK:
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			if next.OpCode() == compiler.OP_MMBINK {
				tag := decodeBytecodeMetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
			}
		}
		return ""

	case compiler.OP_SHLI, compiler.OP_SHRI:
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			if next.OpCode() == compiler.OP_MMBINI {
				tag := decodeBytecodeMetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
			}
		}
		return ""

	// Unary operations
	case compiler.OP_UNM:
		return "unm"
	case compiler.OP_BNOT:
		return "bnot"
	case compiler.OP_LEN:
		return "len"

	// String concatenation
	case compiler.OP_CONCAT:
		return "concat"

	// Comparison operations
	case compiler.OP_EQ, compiler.OP_EQK, compiler.OP_EQI:
		return "eq"
	case compiler.OP_LT, compiler.OP_LTI:
		return "lt"
	case compiler.OP_LE, compiler.OP_LEI:
		return "le"
	case compiler.OP_GTI:
		return "lt" // __lt with reversed operands
	case compiler.OP_GEI:
		return "le" // __le with reversed operands

	// Index operations (when __index metamethod is called)
	case compiler.OP_GETTABLE, compiler.OP_GETI, compiler.OP_GETFIELD:
		return "index"
	case compiler.OP_SETTABLE, compiler.OP_SETI, compiler.OP_SETFIELD:
		return "newindex"

	default:
		return ""
	}
}

// regObjName returns a descriptive string for the value in register `reg`
// at the given PC within a prototype, by scanning backwards through bytecode.
// Returns (name, what) where what is "field", "global", "local", "upvalue",
// or "" if unknown. This matches Lua 5.4's getobjname / kname behavior.
func regObjName(proto *compiler.Proto, pc int, reg int) (string, string) {
	// First check if the register is a local variable at the faulting PC.
	// This handles cases like "local a; a(13)" where the LOADNIL that
	// initialises register 0 occurs before the local's StartPC.
	name := localName(proto, reg, pc)
	if name != "" && name != "?" && !isInternalName(name) {
		return name, "local"
	}

	for i := pc - 1; i >= 0; i-- {
		inst := proto.Code[i]
		op := inst.OpCode()
		a := inst.A()
		if a != reg {
			continue
		}
		// If a forward jump can bypass this instruction and still reach the
		// use site, the assignment is path-dependent (e.g., or/and expressions).
		// Suppress annotation, matching Lua 5.4's filterpc in findsetreg.
		if hasForwardBypass(proto, i, pc) {
			return "", ""
		}
		switch op {
		case compiler.OP_LOADK:
			bx := inst.Bx()
			if bx < len(proto.Constants) && proto.Constants[bx].Type == compiler.ValString {
				return proto.Constants[bx].SVal, "constant"
			}
			return "", ""
		case compiler.OP_LOADKX:
			if i+1 < len(proto.Code) {
				ax := proto.Code[i+1].Ax()
				if ax < len(proto.Constants) && proto.Constants[ax].Type == compiler.ValString {
					return proto.Constants[ax].SVal, "constant"
				}
			}
			return "", ""
		case compiler.OP_GETFIELD:
			c := inst.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				what := "field"
				if localName(proto, inst.B(), i) == "_ENV" || isUpvalEnv(proto, i, inst.B()) {
					what = "global"
				}
				return proto.Constants[c].SVal, what
			}
			return "", ""
		case compiler.OP_GETTABUP:
			c := inst.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				if isUpvalIdx_ENV(proto, inst.B()) {
					return proto.Constants[c].SVal, "global"
				}
				return proto.Constants[c].SVal, "field"
			}
			return "", ""
		case compiler.OP_MOVE:
			reg = inst.B()
			// After following OP_MOVE, check if the new register is a local.
			ln := localName(proto, reg, i)
			if ln != "" && ln != "?" && !isInternalName(ln) {
				return ln, "local"
			}
			continue
		case compiler.OP_GETUPVAL:
			b := inst.B()
			if b < len(proto.Upvalues) && proto.Upvalues[b].Name != "" {
				return proto.Upvalues[b].Name, "upvalue"
			}
			return "", ""
		case compiler.OP_SELF:
			c := inst.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				return proto.Constants[c].SVal, "method"
			}
			return "", ""
		case compiler.OP_GETI:
			return "integer index", "field"
		case compiler.OP_GETTABLE:
			// R[A] = R[B][R[C]] — resolve key name from register C.
			b := inst.B()
			c := inst.C()
			kn := kName(proto, i, c)
			// Check if table is _ENV (global access with high constant index).
			if localName(proto, b, i) == "_ENV" || isUpvalEnv(proto, i, b) {
				if kn != "" {
					return kn, "global"
				}
				return "?", "global"
			}
			// Detect SELF fallback pattern (MOVE base+1,obj + LOADK + GETTABLE).
			if kn != "" && isSelfFallback(proto, i, reg, b) {
				return kn, "method"
			}
			if kn != "" {
				return kn, "field"
			}
			return "?", "field"
		default:
			ln := localName(proto, reg, i)
			if ln != "" && ln != "?" && !isInternalName(ln) {
				return ln, "local"
			}
			return "", ""
		}
	}
	return "", ""
}

// hasForwardBypass reports whether a forward jump before defPC can skip the
// assignment at defPC and still reach usePC. In that case, the assignment is
// path-dependent and should not be treated as the definite source for naming.
func hasForwardBypass(proto *compiler.Proto, defPC, usePC int) bool {
	if defPC <= 0 || usePC <= defPC {
		return false
	}
	for j := defPC - 1; j >= 0; j-- {
		inst := proto.Code[j]
		if inst.OpCode() != compiler.OP_JMP {
			continue
		}
		target := j + 1 + inst.SJ()
		if target > defPC && target <= usePC {
			return true
		}
	}
	return false
}

// isInternalName returns true if the name is a compiler-generated internal
// name (starts with '(' such as "(for state)" or "(for control)").
func isInternalName(name string) bool {
	return len(name) > 0 && name[0] == '('
}

// isUpvalIdx_ENV checks whether the given upvalue index refers to _ENV
// by checking the upvalue's name in the prototype.
func isUpvalIdx_ENV(proto *compiler.Proto, idx int) bool {
	return idx < len(proto.Upvalues) && proto.Upvalues[idx].Name == "_ENV"
}

// isUpvalEnv checks whether the register at the given PC was loaded via
// GETUPVAL from _ENV. Used to distinguish "global" from "field"
// when GETTABLE is the fallback for GETTABUP with large constant indices.
func isUpvalEnv(proto *compiler.Proto, pc int, reg int) bool {
	for i := pc - 1; i >= 0; i-- {
		inst := proto.Code[i]
		if inst.A() != reg {
			continue
		}
		op := inst.OpCode()
		if op == compiler.OP_GETUPVAL && isUpvalIdx_ENV(proto, inst.B()) {
			return true
		}
		// SETFIELD/SETTABLE/SETI use R[A] as the table being written to,
		// not as a destination register. Skip these so we can keep scanning
		// backward to find the GETUPVAL that loaded _ENV into this register.
		if op == compiler.OP_SETFIELD || op == compiler.OP_SETTABLE || op == compiler.OP_SETI {
			continue
		}
		// Also check if the register is a local named _ENV.
		if localName(proto, reg, i) == "_ENV" {
			return true
		}
		return false
	}
	return false
}

// kName resolves a register reference to a constant name at the given PC.
// If register reg was loaded from a constant (via OP_LOADK/OP_LOADKX) and
// the constant is a string, returns that string. Otherwise returns "".
func kName(proto *compiler.Proto, pc int, reg int) string {
	for i := pc - 1; i >= 0; i-- {
		inst := proto.Code[i]
		op := inst.OpCode()
		a := inst.A()
		if a != reg {
			continue
		}
		switch op {
		case compiler.OP_LOADK:
			if hasForwardBypass(proto, i, pc) {
				continue
			}
			bx := inst.Bx()
			if bx < len(proto.Constants) && proto.Constants[bx].Type == compiler.ValString {
				return proto.Constants[bx].SVal
			}
			return ""
		case compiler.OP_LOADKX:
			if hasForwardBypass(proto, i, pc) {
				continue
			}
			if i+1 < len(proto.Code) {
				ax := proto.Code[i+1].Ax()
				if ax < len(proto.Constants) && proto.Constants[ax].Type == compiler.ValString {
					return proto.Constants[ax].SVal
				}
			}
			return ""
		case compiler.OP_MOVE:
			reg = inst.B()
			continue
		default:
			return ""
		}
	}
	return ""
}

// isSelfFallback detects the SELF fallback pattern emitted by the compiler
// when a method name's constant index exceeds MaxArgC. The pattern is:
//
//	MOVE base+1, objReg    // copy self
//	LOADK/LOADKX tmp, "method_name"
//	GETTABLE base, objReg, tmp
//
// Returns true if the instruction at gettablePC is part of this pattern.
func isSelfFallback(proto *compiler.Proto, gettablePC int, base int, objReg int) bool {
	// Look backward past LOADK/LOADKX to find a MOVE writing to base+1
	for j := gettablePC - 1; j >= 0; j-- {
		jInst := proto.Code[j]
		jOp := jInst.OpCode()
		switch jOp {
		case compiler.OP_LOADK, compiler.OP_LOADKX:
			// Skip LOADK that loaded the key into the temp register
			continue
		case compiler.OP_EXTRAARG:
			// Part of LOADKX
			continue
		case compiler.OP_MOVE:
			// Check: MOVE base+1, objReg
			return jInst.A() == base+1 && jInst.B() == objReg
		default:
			return false
		}
	}
	return false
}
