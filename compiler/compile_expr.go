package compiler

import (
	"github.com/iceisfun/golua/ast"
)

// ---------------------------------------------------------------------------
// Expression compilation — main dispatch
// ---------------------------------------------------------------------------

// compileExprToReg compiles expr and ensures its value is in register reg.
func (c *compiler) compileExprToReg(expr ast.Expr, reg int) {
	fs := c.fs

	// Ensure the register is allocated
	if reg >= fs.freeReg {
		fs.freeReg = reg + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}

	switch e := expr.(type) {
	case *ast.NilExpr: // e.g. local x = nil
		fs.emit(ABC(OP_LOADNIL, reg, 0, 0, 0), e.P.Line)

	case *ast.TrueExpr: // e.g. local x = true
		fs.emit(ABC(OP_LOADTRUE, reg, 0, 0, 0), e.P.Line)

	case *ast.FalseExpr: // e.g. local x = false
		fs.emit(ABC(OP_LOADFALSE, reg, 0, 0, 0), e.P.Line)

	case *ast.NumberExpr: // e.g. local x = 42 — LOADI for small ints, LOADK for large
		if e.Value >= -OffsetSBx && e.Value <= OffsetSBx {
			fs.emit(AsBx(OP_LOADI, reg, int(e.Value)), e.P.Line)
		} else {
			k := fs.addConstant(IntValue(e.Value))
			fs.emit(ABx(OP_LOADK, reg, k), e.P.Line)
		}

	case *ast.FloatExpr: // e.g. local x = 3.14 — LOADF for whole floats, LOADK otherwise
		iv := int(e.Value)
		if float64(iv) == e.Value && iv >= -OffsetSBx && iv <= OffsetSBx {
			fs.emit(AsBx(OP_LOADF, reg, iv), e.P.Line)
		} else {
			k := fs.addConstant(FloatValue(e.Value))
			fs.emit(ABx(OP_LOADK, reg, k), e.P.Line)
		}

	case *ast.StringExpr: // e.g. local x = "hello"
		k := fs.addConstant(StringValue(e.Value))
		fs.emit(ABx(OP_LOADK, reg, k), e.P.Line)

	case *ast.NameExpr: // e.g. x — resolves to local, upvalue, or _ENV[x]
		c.compileName(e, reg)

	case *ast.BinopExpr: // e.g. a + b, x .. y, a == b, x and y
		c.compileBinop(e, reg)

	case *ast.UnopExpr: // e.g. -x, not x, #t, ~n
		c.compileUnop(e, reg)

	case *ast.FuncCallExpr: // e.g. f(a, b) — single result in reg
		// When reg < freeReg the target is an existing local. Compiling the
		// call directly at reg would clobber it with the function before
		// arguments that reference it are evaluated. Use a temp register.
		if reg < fs.freeReg {
			tmp := fs.freeReg
			c.compileFuncCall(e, tmp, 2, e.P.Line) // C=2 → 1 result at tmp
			fs.emit(ABC(OP_MOVE, reg, tmp, 0, 0), e.P.Line)
		} else {
			c.compileFuncCall(e, reg, 2, e.P.Line) // C=2 → 1 result
		}

	case *ast.MethodCallExpr: // e.g. obj:method(a) — single result in reg
		if reg < fs.freeReg {
			tmp := fs.freeReg
			c.compileMethodCall(e, tmp, 2, e.P.Line)
			fs.emit(ABC(OP_MOVE, reg, tmp, 0, 0), e.P.Line)
		} else {
			c.compileMethodCall(e, reg, 2, e.P.Line)
		}

	case *ast.FuncExpr: // e.g. function(x) return x end
		protoIdx := c.compileFunc(e, e.P.Line)
		fs.emit(ABx(OP_CLOSURE, reg, protoIdx), e.P.Line)

	case *ast.TableConstructor: // e.g. {1, 2, key="val"}
		c.compileTableConstructor(e, reg)

	case *ast.FieldExpr: // e.g. t.name
		c.compileFieldExpr(e, reg)

	case *ast.IndexExpr: // e.g. t[key]
		c.compileIndexExpr(e, reg)

	case *ast.ParenExpr: // e.g. (expr) — forces single result
		c.compileExprToReg(e.Inner, reg)

	case *ast.VarArgExpr: // e.g. ... — single vararg result
		if !fs.proto.IsVarArg {
			c.error(e, "cannot use '...' outside a vararg function")
		}
		fs.emit(ABC(OP_VARARG, reg, 0, 2, 0), e.P.Line) // C=2 → 1 result

	default:
		c.error(expr, "unhandled expression type %T", expr)
	}
}

// compileExprMultiRet compiles an expression that may return multiple values.
// n is how many results are wanted (0 = all).
func (c *compiler) compileExprMultiRet(expr ast.Expr, n int) {
	fs := c.fs
	base := fs.freeReg

	switch e := expr.(type) {
	case *ast.FuncCallExpr:
		c.compileFuncCall(e, base, 0, e.P.Line) // C=0 → all results
		if n > 0 {
			// Patch CALL's C to n+1
			lastPC := fs.pc() - 1
			inst := fs.proto.Code[lastPC]
			fs.proto.Code[lastPC] = ABC(inst.OpCode(), inst.A(), inst.B(), n+1, 0)
		}
		return

	case *ast.MethodCallExpr:
		c.compileMethodCall(e, base, 0, e.P.Line)
		if n > 0 {
			lastPC := fs.pc() - 1
			inst := fs.proto.Code[lastPC]
			fs.proto.Code[lastPC] = ABC(inst.OpCode(), inst.A(), inst.B(), n+1, 0)
		}
		return

	case *ast.VarArgExpr:
		if !fs.proto.IsVarArg {
			c.error(e, "cannot use '...' outside a vararg function")
		}
		vc := 0
		if n > 0 {
			vc = n + 1
		}
		fs.emit(ABC(OP_VARARG, base, 0, vc, 0), e.P.Line)
		return

	default:
		// Not multi-return — compile normally
		c.compileExprToReg(expr, base)
		// Fill remaining with nil
		if n > 1 {
			fs.emit(ABC(OP_LOADNIL, base+1, n-2, 0, 0), expr.Pos().Line)
		}
	}
}

// ---------------------------------------------------------------------------
// Name lookup (local, upvalue, global)
// ---------------------------------------------------------------------------

// compileName resolves a variable name and emits the appropriate load:
// MOVE for locals, GETUPVAL for upvalues, or GETTABUP for globals (_ENV[name]).
func (c *compiler) compileName(e *ast.NameExpr, reg int) {
	fs := c.fs

	// Local variable
	if localReg, ok := fs.lookupLocal(e.Name); ok {
		if localReg != reg {
			fs.emit(ABC(OP_MOVE, reg, localReg, 0, 0), e.P.Line)
		}
		return
	}

	// If there's a local _ENV, look up via that table instead of upvalues/globals
	if envReg, ok := fs.lookupLocal("_ENV"); ok {
		nameK := fs.stringConstant(e.Name)
		fs.emitGetField(reg, envReg, nameK, e.P.Line)
		return
	}

	// Upvalue
	if uvIdx, ok := c.resolveUpvalue(fs, e.Name); ok {
		fs.emit(ABC(OP_GETUPVAL, reg, uvIdx, 0, 0), e.P.Line)
		return
	}

	// Global: _ENV[name]
	envUV := c.resolveEnv()
	nameK := fs.stringConstant(e.Name)
	fs.emitGetTabUp(reg, envUV, nameK, e.P.Line)
}

// ---------------------------------------------------------------------------
// Binary operations
// ---------------------------------------------------------------------------

// compileBinop compiles a binary operation. Short-circuit operators (and, or),
// concatenation (..), and comparisons are handled by specialized methods.
// Arithmetic and bitwise ops compile both operands into registers and emit
// the operation followed by an MMBIN for metamethod fallback.
func (c *compiler) compileBinop(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	// Handle short-circuit operators
	switch e.Op {
	case "and":
		c.compileAnd(e, reg)
		return
	case "or":
		c.compileOr(e, reg)
		return
	case "..":
		c.compileConcat(e, reg)
		return
	}

	// Handle comparison operators
	switch e.Op {
	case "==", "~=", "<", "<=", ">", ">=":
		c.compileComparison(e, reg)
		return
	}

	// Arithmetic / bitwise — compile both sides into fresh registers so that
	// we never clobber a local that is reused in the same expression.
	// Example: b = a + b — if we compiled left into reg (b's register), it
	// would overwrite b before the right side reads it.
	leftReg := fs.reserveReg()
	c.compileExprToReg(e.Left, leftReg)
	rightReg := fs.reserveReg()
	c.compileExprToReg(e.Right, rightReg)

	var op OpCode
	var mmOp MetamethodTag
	switch e.Op {
	case "+":
		op, mmOp = OP_ADD, TM_ADD
	case "-":
		op, mmOp = OP_SUB, TM_SUB
	case "*":
		op, mmOp = OP_MUL, TM_MUL
	case "%":
		op, mmOp = OP_MOD, TM_MOD
	case "^":
		op, mmOp = OP_POW, TM_POW
	case "/":
		op, mmOp = OP_DIV, TM_DIV
	case "//":
		op, mmOp = OP_IDIV, TM_IDIV
	case "&":
		op, mmOp = OP_BAND, TM_BAND
	case "|":
		op, mmOp = OP_BOR, TM_BOR
	case "~":
		op, mmOp = OP_BXOR, TM_BXOR
	case "<<":
		op, mmOp = OP_SHL, TM_SHL
	case ">>":
		op, mmOp = OP_SHR, TM_SHR
	default:
		c.error(e, "unknown binary operator %q", e.Op)
		return
	}

	fs.emit(ABC(op, reg, leftReg, rightReg, 0), line)
	fs.emit(ABC(OP_MMBIN, leftReg, rightReg, int(mmOp), 0), line)
	fs.freeReg = leftReg
}

// compileConcat flattens a chain of .. operators (e.g. a .. b .. c) into
// consecutive registers and emits a single OP_CONCAT covering all of them.
func (c *compiler) compileConcat(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	// Flatten concat chain: a .. b .. c → [a, b, c] in consecutive regs
	exprs := c.flattenConcat(e)

	// Use freeReg as base to avoid clobbering existing locals/for-loop state.
	// If reg < freeReg, we need to work in temporary space and MOVE result back.
	base := fs.freeReg
	needMove := reg < base

	for i, expr := range exprs {
		c.compileExprToReg(expr, base+i)
		if base+i >= fs.freeReg {
			fs.freeReg = base + i + 1
			if fs.freeReg > fs.maxReg {
				fs.maxReg = fs.freeReg
			}
		}
	}

	fs.emit(ABC(OP_CONCAT, base, len(exprs), 0, 0), line)

	if needMove {
		fs.emit(ABC(OP_MOVE, reg, base, 0, 0), line)
	}

	fs.freeReg = base + 1
}

// flattenConcat recursively collects all operands of a .. chain into a flat slice.
func (c *compiler) flattenConcat(e ast.Expr) []ast.Expr {
	if binop, ok := e.(*ast.BinopExpr); ok && binop.Op == ".." {
		left := c.flattenConcat(binop.Left)
		right := c.flattenConcat(binop.Right)
		return append(left, right...)
	}
	return []ast.Expr{e}
}

// compileComparison compiles == ~= < <= > >= into a comparison instruction
// followed by conditional jumps that produce a boolean result in reg.
func (c *compiler) compileComparison(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	leftReg := fs.freeReg
	fs.reserveReg()
	c.compileExprToReg(e.Left, leftReg)
	rightReg := fs.reserveReg()
	c.compileExprToReg(e.Right, rightReg)

	var op OpCode
	k := 1 // test sense
	switch e.Op {
	case "==":
		op = OP_EQ
		k = 0
	case "~=":
		op = OP_EQ
		k = 1
	case "<":
		op = OP_LT
		k = 0
	case "<=":
		op = OP_LE
		k = 0
	case ">":
		op = OP_LT
		k = 0
		leftReg, rightReg = rightReg, leftReg // swap
	case ">=":
		op = OP_LE
		k = 0
		leftReg, rightReg = rightReg, leftReg // swap
	}

	// comparison + conditional jump → boolean
	fs.emit(ABC(op, leftReg, rightReg, 0, k), line)
	jmpFalse := fs.emitJump(line) // skip next if comparison fails
	fs.emit(ABC(OP_LOADTRUE, reg, 0, 0, 0), line)
	jmpEnd := fs.emitJump(line)
	c.patchJump(jmpFalse)
	fs.emit(ABC(OP_LOADFALSE, reg, 0, 0, 0), line)
	c.patchJump(jmpEnd)

	fs.freeReg = leftReg
}

// ---------------------------------------------------------------------------
// Logical and/or
// ---------------------------------------------------------------------------

// compileAnd compiles "a and b" — short-circuits to a if a is falsy.
// Uses a temporary register for the left operand so that OP_TESTSET does not
// clobber a local variable that shares the destination register (e.g.
// "max = false and 99 or max" where max is both destination and operand).
func (c *compiler) compileAnd(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	tmp := fs.reserveReg()
	c.compileExprToReg(e.Left, tmp)
	fs.emit(ABC(OP_TESTSET, reg, tmp, 0, 0), line) // skip if falsy, keep value
	jmp := fs.emitJump(line)                        // jump to end (short-circuit)
	c.compileExprToReg(e.Right, reg)
	c.patchJump(jmp)
	fs.freeReg = tmp
}

// compileOr compiles "a or b" — short-circuits to a if a is truthy.
// Uses a temporary register for the left operand so that OP_TESTSET does not
// clobber a local variable that shares the destination register.
func (c *compiler) compileOr(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	tmp := fs.reserveReg()
	c.compileExprToReg(e.Left, tmp)
	fs.emit(ABC(OP_TESTSET, reg, tmp, 0, 1), line) // skip if truthy, keep value
	jmp := fs.emitJump(line)                        // jump to end (short-circuit)
	c.compileExprToReg(e.Right, reg)
	c.patchJump(jmp)
	fs.freeReg = tmp
}

// ---------------------------------------------------------------------------
// Unary operations
// ---------------------------------------------------------------------------

// compileUnop compiles a unary operator: -x (UNM), not x (NOT), #t (LEN), ~n (BNOT).
func (c *compiler) compileUnop(e *ast.UnopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	// Compile operand into a (possibly temporary) register
	opReg := reg
	c.compileExprToReg(e.Operand, opReg)

	switch e.Op {
	case "-":
		fs.emit(ABC(OP_UNM, reg, opReg, 0, 0), line)
	case "not":
		fs.emit(ABC(OP_NOT, reg, opReg, 0, 0), line)
	case "#":
		fs.emit(ABC(OP_LEN, reg, opReg, 0, 0), line)
	case "~":
		fs.emit(ABC(OP_BNOT, reg, opReg, 0, 0), line)
	default:
		c.error(e, "unknown unary operator %q", e.Op)
	}
}

// ---------------------------------------------------------------------------
// Function calls
// ---------------------------------------------------------------------------

// compileFuncCall compiles a function call expression.
// base: first register for the call frame
// nResults: number of results wanted (0 = all, 1 = none, 2 = one, etc.)
// Lua convention: CALL A B C → C = nResults + 1 (0 means all)
func (c *compiler) compileFuncCall(e *ast.FuncCallExpr, base int, nResults int, line int) {
	fs := c.fs

	// Ensure base is allocated
	if base >= fs.freeReg {
		fs.freeReg = base + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}

	// Function goes into base
	c.compileExprToReg(e.Func, base)

	// Arguments into base+1, base+2, ...
	nArgs := len(e.Args)
	for i, arg := range e.Args {
		if i == nArgs-1 && isMultiRet(arg) {
			c.compileExprMultiRet(arg, 0) // open call
			fs.emit(ABC(OP_CALL, base, 0, nResults, 0), line) // B=0 means top
			return
		}
		argReg := base + 1 + i
		c.compileExprToReg(arg, argReg)
		// Reset freeReg to reclaim any temporaries used by the expression
		// (e.g., inner function call arguments). Without this, the last
		// multi-ret argument would start at an inflated freeReg, leaving
		// a gap of stale values that the outer B=0 call picks up.
		fs.freeReg = argReg + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}

	fs.emit(ABC(OP_CALL, base, nArgs+1, nResults, 0), line)
}

// compileMethodCall compiles obj:method(args). Emits SELF to load the method
// and self reference, then CALL. nResults follows the same convention as compileFuncCall.
func (c *compiler) compileMethodCall(e *ast.MethodCallExpr, base int, nResults int, line int) {
	fs := c.fs

	if base >= fs.freeReg {
		fs.freeReg = base + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}

	// SELF: R[base+1] = R[obj]; R[base] = R[obj][method]
	objReg := fs.reserveReg()
	c.compileExprToReg(e.Object, objReg)
	methodK := fs.stringConstant(e.Method)
	fs.emitSelf(base, objReg, methodK, line)

	// Make sure base+1 is allocated
	if base+1 >= fs.freeReg {
		fs.freeReg = base + 2
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}

	// Arguments start at base+2
	nArgs := len(e.Args)
	for i, arg := range e.Args {
		if i == nArgs-1 && isMultiRet(arg) {
			c.compileExprMultiRet(arg, 0)
			fs.emit(ABC(OP_CALL, base, 0, nResults, 0), line)
			return
		}
		argReg := base + 2 + i
		c.compileExprToReg(arg, argReg)
		// Reset freeReg to reclaim temporaries (same fix as compileFuncCall)
		fs.freeReg = argReg + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}

	// +1 for self
	fs.emit(ABC(OP_CALL, base, nArgs+2, nResults, 0), line)
	fs.freeReg = objReg
}

// ---------------------------------------------------------------------------
// Table constructors
// ---------------------------------------------------------------------------

// compileTableConstructor compiles {1, 2, key="val"} — emits NEWTABLE,
// then SETLIST for array entries and SETFIELD/SETTABLE for hash entries.
func (c *compiler) compileTableConstructor(e *ast.TableConstructor, reg int) {
	fs := c.fs
	line := e.P.Line

	// Count array and hash parts
	nArr := 0
	nHash := 0
	for _, f := range e.Fields {
		if f.Key == nil {
			nArr++
		} else {
			nHash++
		}
	}

	// NEWTABLE with size hints
	hashLog := 0
	if nHash > 0 {
		hashLog = intLog2(nHash) + 1
	}
	// Use IvABC format: vB (6 bits) = hash log, vC (10 bits) = array size
	vB := hashLog
	vC := nArr
	k := 0
	if vC > 0x3FF {
		// Need EXTRAARG for array size
		k = 1
		vC = 0
	}
	inst := Instruction(uint32(OP_NEWTABLE)<<PosOP |
		uint32(reg)<<PosA |
		uint32(vB)<<16 |
		uint32(vC)<<22 |
		uint32(k)<<PosK)
	fs.emit(inst, line)

	if k == 1 {
		fs.emit(Ax(OP_EXTRAARG, nArr), line)
	}

	// Find the last array-style field to check if it's multi-return
	lastArrayIdx := -1
	for i, f := range e.Fields {
		if f.Key == nil {
			lastArrayIdx = i
		}
	}

	// Fill table fields
	arrIdx := 0
	pendingList := 0 // number of pending SETLIST items
	for i, f := range e.Fields {
		if f.Key == nil {
			// Array-style field
			arrIdx++
			pendingList++
			arrReg := reg + pendingList
			if arrReg >= fs.freeReg {
				fs.freeReg = arrReg + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			}

			// Check if this is the last array element and it's a multi-return expression
			if i == lastArrayIdx && isMultiRet(f.Value) {
				// Set freeReg to arrReg so compileExprMultiRet uses the correct base
				fs.freeReg = arrReg
				// Compile with all results
				c.compileExprMultiRet(f.Value, 0) // 0 = all results
				// Emit SETLIST with count=0 to capture all values up to top
				c.emitSetList(reg, 0, arrIdx-pendingList+1, line)
				pendingList = 0
				fs.freeReg = reg + 1
				continue
			}

			c.compileExprToReg(f.Value, arrReg)

			// Flush in batches of 50 (like Lua's LFIELDS_PER_FLUSH)
			if pendingList >= 50 {
				c.emitSetList(reg, pendingList, arrIdx-pendingList+1, line)
				pendingList = 0
				fs.freeReg = reg + 1
			}
		} else {
			// Hash-style field
			switch key := f.Key.(type) {
			case *ast.StringExpr:
				valReg := fs.reserveReg()
				c.compileExprToReg(f.Value, valReg)
				kIdx := fs.stringConstant(key.Value)
				fs.emitSetField(reg, kIdx, valReg, line)
				fs.freeReg = valReg
			default:
				keyReg := fs.reserveReg()
				c.compileExprToReg(f.Key, keyReg)
				valReg := fs.reserveReg()
				c.compileExprToReg(f.Value, valReg)
				fs.emit(ABC(OP_SETTABLE, reg, keyReg, valReg, 0), line)
				fs.freeReg = keyReg
			}
		}
	}

	// Flush remaining array items
	if pendingList > 0 {
		c.emitSetList(reg, pendingList, arrIdx-pendingList+1, line)
	}

	fs.freeReg = reg + 1
}

// emitSetList emits OP_SETLIST to flush pending array entries into a table.
// count is the number of entries (0 = up to stack top), offset is the 1-based starting index.
func (c *compiler) emitSetList(tableReg, count, offset int, line int) {
	fs := c.fs
	// IvABC format: vB = count, vC = offset
	vB := count
	vC := offset
	k := 0
	if vC > 0x3FF {
		k = 1
		vC = 0
	}
	inst := Instruction(uint32(OP_SETLIST)<<PosOP |
		uint32(tableReg)<<PosA |
		uint32(vB)<<16 |
		uint32(vC)<<22 |
		uint32(k)<<PosK)
	fs.emit(inst, line)
	if k == 1 {
		fs.emit(Ax(OP_EXTRAARG, offset), line)
	}
}

// ---------------------------------------------------------------------------
// Field and index access
// ---------------------------------------------------------------------------

// compileFieldExpr compiles t.name into GETFIELD (or GETTABLE for large constant indices).
func (c *compiler) compileFieldExpr(e *ast.FieldExpr, reg int) {
	fs := c.fs
	tableReg := fs.reserveReg()
	c.compileExprToReg(e.Table, tableReg)
	fieldK := fs.stringConstant(e.Field)
	fs.emitGetField(reg, tableReg, fieldK, e.P.Line)
	fs.freeReg = tableReg
}

// compileIndexExpr compiles t[key] into GETTABLE.
func (c *compiler) compileIndexExpr(e *ast.IndexExpr, reg int) {
	fs := c.fs
	tableReg := fs.reserveReg()
	c.compileExprToReg(e.Table, tableReg)
	keyReg := fs.reserveReg()
	c.compileExprToReg(e.Key, keyReg)
	fs.emit(ABC(OP_GETTABLE, reg, tableReg, keyReg, 0), e.P.Line)
	fs.freeReg = tableReg
}
