package compiler

import (
	"fmt"
	"math"

	"github.com/iceisfun/golua/v2/ast"
)

// exprEndLine returns the line of the last token in an expression.
// For binary expressions, this is the end line of the right operand.
// For index/field expressions, this is the end line of the key/field.
// For simple expressions (names, numbers), this is the expression's own line.
// This is used to match Lua 5.4's parser line tracking where the "current
// line" after parsing an expression reflects the last token parsed.
func exprEndLine(e ast.Expr) int {
	switch expr := e.(type) {
	case *ast.BinopExpr:
		return exprEndLine(expr.Right)
	case *ast.UnopExpr:
		return exprEndLine(expr.Operand)
	case *ast.IndexExpr:
		return exprEndLine(expr.Key)
	case *ast.FieldExpr:
		// field name is on the same line as the dot
		return expr.P.Line
	case *ast.FuncCallExpr:
		if len(expr.Args) > 0 {
			return exprEndLine(expr.Args[len(expr.Args)-1])
		}
		return expr.P.Line
	case *ast.MethodCallExpr:
		if len(expr.Args) > 0 {
			return exprEndLine(expr.Args[len(expr.Args)-1])
		}
		return expr.P.Line
	case *ast.ParenExpr:
		return exprEndLine(expr.Inner)
	}
	return e.Pos().Line
}

// exprNear returns a short "near" token string from an AST expression,
// for use in error messages. Returns "" if no meaningful token can be
// extracted. Matches Lua 5.4's behavior of showing the current token.
func exprNear(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.NameExpr:
		return e.Name
	case *ast.NumberExpr:
		return e.Raw
	case *ast.FloatExpr:
		return e.Raw
	case *ast.StringExpr:
		return e.Value
	case *ast.VarArgExpr:
		return "..."
	case *ast.NilExpr:
		return "nil"
	case *ast.TrueExpr:
		return "true"
	case *ast.FalseExpr:
		return "false"
	case *ast.FuncCallExpr:
		return exprNear(e.Func)
	case *ast.MethodCallExpr:
		return e.Method
	case *ast.FieldExpr:
		return e.Field
	case *ast.IndexExpr:
		return exprNear(e.Key)
	case *ast.BinopExpr:
		return exprNear(e.Right)
	case *ast.UnopExpr:
		return fmt.Sprintf("%s", e.Op)
	}
	return ""
}

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
		if fs.freeReg > fs.c.limits.MaxRegs {
			// Lua 5.5 reworded this from the 5.4 "function or expression needs
			// too many registers" to the errorlimit() form
			// "too many registers (limit is N) in main function/in function at
			// line N[ near '<token>']".
			fs.limitError("registers", fs.c.limits.MaxRegs, expr.Pos().Line, exprNear(expr))
		}
	}

	switch e := expr.(type) {
	case *ast.NilExpr: // e.g. local x = nil
		fs.emit(ABC(OP_LOADNIL, reg, 0, 0, 0), e.P.Line)

	case *ast.TrueExpr: // e.g. local x = true
		fs.emit(ABC(OP_LOADTRUE, reg, 0, 0, 0), e.P.Line)

	case *ast.FalseExpr: // e.g. local x = false
		fs.emit(ABC(OP_LOADFALSE, reg, 0, 0, 0), e.P.Line)

	case *ast.NumberExpr: // e.g. local x = 42 — LOADI for small ints, LOADK/LOADKX for large
		if e.Value >= -OffsetSBx && e.Value <= OffsetSBx {
			fs.emit(AsBx(OP_LOADI, reg, int(e.Value)), e.P.Line)
		} else {
			k := fs.addConstant(IntValue(e.Value))
			fs.loadConstant(reg, k, e.P.Line)
		}

	case *ast.FloatExpr: // e.g. local x = 3.14 — LOADF for whole floats, LOADK/LOADKX otherwise
		iv := int(e.Value)
		if float64(iv) == e.Value && iv >= -OffsetSBx && iv <= OffsetSBx && !math.Signbit(e.Value) {
			fs.emit(AsBx(OP_LOADF, reg, iv), e.P.Line)
		} else {
			k := fs.addConstant(FloatValue(e.Value))
			fs.loadConstant(reg, k, e.P.Line)
		}

	case *ast.StringExpr: // e.g. local x = "hello"
		k := fs.addConstant(StringValue(c.internString(e.Value)))
		fs.loadConstant(reg, k, e.P.Line)

	case *ast.NameExpr: // e.g. x — resolves to local, upvalue, or _ENV[x]
		c.compileName(e, reg)

	case *ast.BinopExpr: // e.g. a + b, x .. y, a == b, x and y
		c.compileBinop(e, reg)

	case *ast.UnopExpr: // e.g. -x, not x, #t, ~n
		c.compileUnop(e, reg)

	case *ast.FuncCallExpr: // e.g. f(a, b) — single result in reg
		// When reg holds a live local, compiling the call directly at reg
		// would clobber it with the function before arguments that reference
		// it are evaluated — use a temp register. A scratch target compiles
		// in place (the call only writes reg and up, where nothing is live),
		// so chains like t.a().a()... reuse one register per link.
		if reg < fs.regTop() {
			tmp := fs.freeReg
			c.compileFuncCall(e, tmp, 2, e.P.Line) // C=2 → 1 result at tmp
			fs.emit(ABC(OP_MOVE, reg, tmp, 0, 0), e.P.Line)
		} else {
			c.compileFuncCall(e, reg, 2, e.P.Line) // C=2 → 1 result
		}

	case *ast.MethodCallExpr: // e.g. obj:method(a) — single result in reg
		// Same live-local guard as FuncCallExpr above.
		if reg < fs.regTop() {
			tmp := fs.freeReg
			c.compileMethodCall(e, tmp, 2, e.P.Line)
			fs.emit(ABC(OP_MOVE, reg, tmp, 0, 0), e.P.Line)
		} else {
			c.compileMethodCall(e, reg, 2, e.P.Line)
		}

	case *ast.FuncExpr: // e.g. function(x) return x end
		protoIdx := c.compileFunc(e, e.P.Line)
		closureLine := e.EndLine
		if closureLine == 0 {
			closureLine = e.P.Line
		}
		fs.emit(ABx(OP_CLOSURE, reg, protoIdx), closureLine)

	case *ast.TableConstructor: // e.g. {1, 2, key="val"}
		// When reg holds a live local (reassignment target), compiling the
		// constructor directly at reg would clobber it with NEWTABLE before
		// expressions that reference it (e.g. t = {table.unpack(t)}) are
		// evaluated, and scratch space at reg+1, reg+2, ... could clobber
		// other locals. Use a temp register and move. A scratch target
		// (reg >= regTop) compiles in place: everything above it is dead,
		// and the direct path costs one register per constructor nesting
		// level like reference Lua instead of two.
		if reg < fs.regTop() {
			tmp := fs.freeReg
			c.compileTableConstructor(e, tmp)
			fs.emit(ABC(OP_MOVE, reg, tmp, 0, 0), e.P.Line)
			fs.freeReg = tmp + 1
		} else {
			c.compileTableConstructor(e, reg)
		}

	case *ast.FieldExpr: // e.g. t.name
		c.compileFieldExpr(e, reg)

	case *ast.IndexExpr: // e.g. t[key]
		c.compileIndexExpr(e, reg)

	case *ast.ParenExpr: // e.g. (expr) — forces single result
		c.compileExprToReg(e.Inner, reg)

	case *ast.VarArgExpr: // e.g. ... — single vararg result
		if !fs.proto.IsVarArg {
			c.error(e, "cannot use '...' outside a vararg function near '...'")
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
			c.error(e, "cannot use '...' outside a vararg function near '...'")
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

	// Inlined `<const>` local — emit the constant directly. Search the
	// current function's locals first; if not found, walk enclosing
	// functions (inlined consts are inlined across closure boundaries
	// too — they consume no upvalue slot).
	if val, ok := lookupInlinedAny(fs, e.Name); ok {
		c.emitInlinedConst(reg, val, e.P.Line)
		return
	}

	// Local variable
	if localReg, ok := fs.lookupLocal(e.Name); ok {
		if localReg != reg {
			fs.emit(ABC(OP_MOVE, reg, localReg, 0, 0), e.P.Line)
		}
		return
	}

	// Upvalue — check before local _ENV so that captured locals from
	// enclosing functions are not shadowed by _ENV table lookups.
	if uvIdx, ok := c.resolveUpvalue(fs, e.Name); ok {
		fs.emit(ABC(OP_GETUPVAL, reg, uvIdx, 0, 0), e.P.Line)
		return
	}

	// If there's a local _ENV, look up via that table instead of globals
	if envReg, ok := fs.lookupLocal(envUpvalueName); ok {
		c.checkGlobalRead(e.Name, e)
		nameK := fs.stringConstant(e.Name)
		fs.emitGetField(reg, envReg, nameK, e.P.Line)
		return
	}

	// Global: _ENV[name]
	c.checkGlobalRead(e.Name, e)
	envUV := c.resolveEnv()
	nameK := fs.stringConstant(e.Name)
	fs.emitGetTabUp(reg, envUV, nameK, e.P.Line)
}

// ---------------------------------------------------------------------------
// Binary operations
// ---------------------------------------------------------------------------

// operandToReg resolves expr to a register usable as an operand (B field).
// If expr is a plain local variable already living in a register, that
// register is returned directly with tmp=false (no MOVE, no reservation) —
// matching reference Lua's exp2anyreg, which reads in-register locals in
// place. Otherwise a fresh temp is reserved, expr is compiled into it, and
// tmp=true is returned so the caller frees it afterward. Reading the local
// in place avoids an extra MOVE and, crucially, avoids reserving a temp that
// would push a following call's target register up by one and leave a stale
// "(temporary)" slot visible to debug.getlocal.
func (c *compiler) operandToReg(expr ast.Expr) (reg int, tmp bool) {
	fs := c.fs
	if name, ok := unparen(expr).(*ast.NameExpr); ok {
		if _, inlined := lookupInlinedAny(fs, name.Name); !inlined {
			if localReg, ok := fs.lookupLocal(name.Name); ok {
				return localReg, false
			}
		}
	}
	reg = fs.reserveReg()
	c.compileExprToReg(expr, reg)
	return reg, true
}

// plainLocalReg returns the register of expr when it is a plain, non-inlined
// local variable already living in a register (readable in place as an
// instruction operand), matching operandToReg's in-place case. Returns ok=false
// otherwise. Used to read a binary/comparison left operand live at execution
// time instead of snapshotting it into a temp with a MOVE.
func plainLocalReg(fs *funcState, expr ast.Expr) (int, bool) {
	if name, ok := unparen(expr).(*ast.NameExpr); ok {
		if _, inlined := lookupInlinedAny(fs, name.Name); !inlined {
			if localReg, ok := fs.lookupLocal(name.Name); ok {
				return localReg, true
			}
		}
	}
	return 0, false
}

// unparen strips redundant parentheses from an expression. In reference Lua
// '(' expr ')' compiles to nothing but luaK_dischargevars: parentheses are
// inert apart from truncating a multi-value expression to a single value, and
// in particular a parenthesised local stays in its own register and is read
// live by the instruction that uses it. Register-level decisions must
// therefore see through them, or `(a) + f()` snapshots `a` into a temp before
// f() runs and `(t).k = f()` stores into the wrong table. Truncation is
// unaffected: callers use the unwrapped node only to recognise a bare name,
// which is always single-valued.
func unparen(expr ast.Expr) ast.Expr {
	for {
		p, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = p.Inner
	}
}

// smallIntConst checks whether an expression is a small integer constant
// that fits in the signed C field (sC range: -OffsetSC to +OffsetSC, i.e.
// -127 to +127). If so, it returns the value and true.
func smallIntConst(e ast.Expr) (int, bool) {
	n, ok := e.(*ast.NumberExpr)
	if !ok {
		return 0, false
	}
	if n.Value >= -int64(OffsetSC) && n.Value <= int64(OffsetSC) {
		return int(n.Value), true
	}
	return 0, false
}

// foldArith attempts compile-time constant folding for a binary expression.
// If both operands are numeric constants (integer or float literals, or
// recursively foldable sub-expressions), the result is computed at compile
// time and returned as an AST expression node. Returns nil if folding is
// not possible (e.g., non-constant operand, division by zero, etc.).
//
// This matches Lua 5.4's compile-time constant folding, which allows
// long chains like 1+1+1+...+1 to compile into a single constant without
// consuming registers.
func (c *compiler) foldArith(e *ast.BinopExpr) ast.Expr {
	// Extract numeric values from both sides, recursing into sub-expressions.
	lIsInt, lInt, lFloat, lOk := c.numericValue(e.Left)
	if !lOk {
		return nil
	}
	rIsInt, rInt, rFloat, rOk := c.numericValue(e.Right)
	if !rOk {
		return nil
	}

	// Both integers: try integer arithmetic.
	if lIsInt && rIsInt {
		if result, ok := foldIntOp(e.Op, lInt, rInt); ok {
			return &ast.NumberExpr{P: e.P, Value: result, Raw: fmt.Sprintf("%d", result)}
		}
		// For / and ^, fall through to float path.
	}

	// floatFold returns a folded float result, but declines to fold when the
	// result is NaN or 0.0 — matching reference Lua's constfolding() in
	// lcode.c: "folds neither NaN nor 0.0 (to avoid problems with -0.0)".
	// IEEE means runtime arithmetic must produce the correct sign of zero
	// (e.g. (-0.0) + 0.0 = +0.0 vs (-0.0) - 0.0 = -0.0); folding at compile
	// time would freeze a single sign and lose that distinction.
	floatFold := func(r float64) ast.Expr {
		if math.IsNaN(r) || r == 0 {
			return nil
		}
		return &ast.FloatExpr{P: e.P, Value: r, Raw: fmt.Sprintf("%g", r)}
	}

	// Float path (at least one float, or int-only ops that produce float).
	switch e.Op {
	case "+":
		return floatFold(lFloat + rFloat)
	case "-":
		return floatFold(lFloat - rFloat)
	case "*":
		return floatFold(lFloat * rFloat)
	case "/":
		if rFloat == 0 {
			return nil // leave division by zero to runtime for correct NaN sign
		}
		return floatFold(lFloat / rFloat)
	case "//":
		if rFloat == 0 {
			return nil
		}
		return floatFold(math.Floor(lFloat / rFloat))
	case "%":
		if rFloat == 0 {
			return nil
		}
		r := math.Mod(lFloat, rFloat)
		if r != 0 && (r < 0) != (rFloat < 0) {
			r += rFloat
		}
		return floatFold(r)
	case "^":
		r := powWithSubnormalFix(lFloat, rFloat)
		if math.IsNaN(r) || math.IsInf(r, 0) {
			return nil // leave edge cases to runtime for correct NaN/Inf sign
		}
		return floatFold(r)
	case "&", "|", "~", "<<", ">>":
		// Bitwise ops require integers; if either is float, no fold.
		return nil
	}
	return nil
}

// foldIntOp performs integer arithmetic for constant folding.
// Returns the result and true if the operation can be folded as an integer.
func foldIntOp(op string, a, b int64) (int64, bool) {
	switch op {
	case "+":
		return a + b, true // wraps on overflow, matching Lua 5.4
	case "-":
		return a - b, true
	case "*":
		return a * b, true
	case "//":
		if b == 0 {
			return 0, false
		}
		q := a / b
		if (a^b) < 0 && q*b != a {
			q--
		}
		return q, true
	case "%":
		if b == 0 {
			return 0, false
		}
		r := a % b
		if r != 0 && (r^b) < 0 {
			r += b
		}
		return r, true
	case "&":
		return a & b, true
	case "|":
		return a | b, true
	case "~":
		return a ^ b, true
	case "<<":
		if b >= 64 || b <= -64 {
			return 0, true
		}
		if b < 0 {
			return int64(uint64(a) >> uint64(-b)), true
		}
		return int64(uint64(a) << uint64(b)), true
	case ">>":
		if b >= 64 || b <= -64 {
			return 0, true
		}
		if b < 0 {
			return int64(uint64(a) << uint64(-b)), true
		}
		return int64(uint64(a) >> uint64(b)), true
	}
	// / and ^ always produce floats in Lua
	return 0, false
}

// numericValue extracts a numeric value from an expression, recursively
// folding sub-expressions. Returns (isInteger, intVal, floatVal, ok).
// When isInteger is true, both intVal and floatVal are set (floatVal =
// float64(intVal)) for use in mixed-type arithmetic.
func (c *compiler) numericValue(e ast.Expr) (bool, int64, float64, bool) {
	switch n := e.(type) {
	case *ast.NumberExpr:
		return true, n.Value, float64(n.Value), true
	case *ast.FloatExpr:
		return false, 0, n.Value, true
	case *ast.BinopExpr:
		if r, ok := c.foldMemo[n]; ok {
			return r.isInt, r.iv, r.fv, r.ok
		}
		isInt, iv, fv, ok := c.numericValueBinop(n)
		c.foldMemo[n] = foldResult{isInt, iv, fv, ok}
		return isInt, iv, fv, ok
	case *ast.UnopExpr:
		if r, ok := c.foldMemo[n]; ok {
			return r.isInt, r.iv, r.fv, r.ok
		}
		isInt, iv, fv, ok := false, int64(0), 0.0, false
		if folded := c.foldUnaryArith(n); folded != nil {
			isInt, iv, fv, ok = c.numericValue(folded)
		}
		c.foldMemo[n] = foldResult{isInt, iv, fv, ok}
		return isInt, iv, fv, ok
	}
	return false, 0, 0, false
}

// numericValueBinop computes the folded numeric value of a binop node (the
// caller memoizes the result).
func (c *compiler) numericValueBinop(n *ast.BinopExpr) (bool, int64, float64, bool) {
	folded := c.foldArith(n)
	if folded == nil {
		return false, 0, 0, false
	}
	return c.numericValue(folded)
}

// lookupInlinedAny searches the current function and all enclosing
// functions for an active inlined `<const>` local with the given name.
// Returns the constant value and ok=true on success. Lua 5.5 inlines
// `<const>` locals across closure boundaries (they take no upvalue
// slot in inner functions), so the search walks up `fs.parent`.
func lookupInlinedAny(fs *funcState, name string) (Value, bool) {
	for cur := fs; cur != nil; cur = cur.parent {
		if v, ok := cur.lookupInlined(name); ok {
			return v, true
		}
		// If a regular local with that name shadows in any enclosing
		// scope, stop (it would be the resolution target, not the
		// inlined binding).
		if _, ok := cur.lookupLocal(name); ok {
			return Value{}, false
		}
	}
	return Value{}, false
}

// emitInlinedConst emits the bytecode to materialize a Value into reg,
// matching the codegen used for the corresponding literal expression.
// Used at use-sites of inlined `<const>` locals.
func (c *compiler) emitInlinedConst(reg int, v Value, line int) {
	fs := c.fs
	switch v.Type {
	case ValNil:
		fs.emit(ABC(OP_LOADNIL, reg, 0, 0, 0), line)
	case ValTrue:
		fs.emit(ABC(OP_LOADTRUE, reg, 0, 0, 0), line)
	case ValFalse:
		fs.emit(ABC(OP_LOADFALSE, reg, 0, 0, 0), line)
	case ValInt:
		if v.IVal >= -OffsetSBx && v.IVal <= OffsetSBx {
			fs.emit(AsBx(OP_LOADI, reg, int(v.IVal)), line)
		} else {
			k := fs.addConstant(v)
			fs.loadConstant(reg, k, line)
		}
	case ValFloat:
		iv := int(v.FVal)
		if float64(iv) == v.FVal && iv >= -OffsetSBx && iv <= OffsetSBx && !math.Signbit(v.FVal) {
			fs.emit(AsBx(OP_LOADF, reg, iv), line)
		} else {
			k := fs.addConstant(v)
			fs.loadConstant(reg, k, line)
		}
	case ValString:
		k := fs.addConstant(StringValue(c.internString(v.SVal)))
		fs.loadConstant(reg, k, line)
	default:
		// Unreachable: tryFoldConstScalar only produces scalar Values.
		fs.emit(ABC(OP_LOADNIL, reg, 0, 0, 0), line)
	}
}

// tryFoldConstScalar attempts to evaluate expr to a compile-time scalar
// constant suitable for inlining a `<const>` local. Returns the resulting
// Value (typed nil/bool/int/float/string) and ok=true on success. Mirrors
// Lua 5.5's reference compiler which inlines `<const>` locals whose
// initializer is a literal nil, true, false, integer, float, or string,
// or any compile-time-foldable arithmetic over those (matching the
// `vkisinreg`/constant test in `lparser.c`).
//
// Function and table initializers are intentionally not inlined: even
// `<const>` locals bound to those keep a real register and a debug entry.
func (c *compiler) tryFoldConstScalar(expr ast.Expr) (Value, bool) {
	switch e := expr.(type) {
	case *ast.NilExpr:
		return NilValue(), true
	case *ast.TrueExpr:
		return BoolValue(true), true
	case *ast.FalseExpr:
		return BoolValue(false), true
	case *ast.NumberExpr:
		return IntValue(e.Value), true
	case *ast.FloatExpr:
		return FloatValue(e.Value), true
	case *ast.StringExpr:
		return StringValue(e.Value), true
	case *ast.ParenExpr:
		return c.tryFoldConstScalar(e.Inner)
	case *ast.UnopExpr:
		// Numeric unary folds (-x, ~x).
		if folded := c.foldUnaryArith(e); folded != nil {
			return c.tryFoldConstScalar(folded)
		}
		return Value{}, false
	case *ast.BinopExpr:
		// Arithmetic/bitwise binary folds (1+2, 3*4, 0xff&0x0f, ...).
		if folded := c.foldArith(e); folded != nil {
			return c.tryFoldConstScalar(folded)
		}
		return Value{}, false
	}
	return Value{}, false
}

// foldUnaryArith attempts constant folding for unary minus and bitwise not.
func (c *compiler) foldUnaryArith(e *ast.UnopExpr) ast.Expr {
	isInt, iv, fv, ok := c.numericValue(e.Operand)
	if !ok {
		return nil
	}
	switch e.Op {
	case "-":
		if isInt {
			r := -iv
			return &ast.NumberExpr{P: e.P, Value: r, Raw: fmt.Sprintf("%d", r)}
		}
		r := -fv
		return &ast.FloatExpr{P: e.P, Value: r, Raw: fmt.Sprintf("%g", r)}
	case "~":
		if !isInt {
			return nil
		}
		r := ^iv
		return &ast.NumberExpr{P: e.P, Value: r, Raw: fmt.Sprintf("%d", r)}
	}
	return nil
}

// compileBinop compiles a binary operation. Short-circuit operators (and, or),
// concatenation (..), and comparisons are handled by specialized methods.
// Arithmetic and bitwise ops compile both operands into registers and emit
// the operation followed by an MMBIN for metamethod fallback.
//
// When the "+" operator has a small integer constant on one side, the
// compiler emits OP_ADDI + OP_MMBINI instead of OP_LOADI + OP_ADD +
// OP_MMBIN, saving one instruction per addition. This matches reference
// Lua 5.4's codegen.
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

	// Constant folding: if both operands are compile-time constants,
	// evaluate the result now and emit a single load instruction.
	// This prevents long chains like 1+1+1+...+1 from exhausting registers.
	if folded := c.foldArith(e); folded != nil {
		c.compileExprToReg(folded, reg)
		return
	}

	// ADDI optimization: when "+" has a small integer constant operand,
	// emit OP_ADDI + OP_MMBINI (2 instructions) instead of OP_LOADI +
	// OP_ADD + OP_MMBIN (3 instructions).
	if e.Op == "+" {
		if imm, ok := smallIntConst(e.Right); ok {
			// a + imm  →  ADDI reg, leftReg, imm; MMBINI leftReg, imm, TM_ADD, 0
			leftReg, tmp := c.operandToReg(e.Left)
			if tmp {
				// Only fix the discharged-operand line when operandToReg
				// actually emitted an instruction. If e.Left was a local read
				// in place, nothing was emitted and fixDischargedLine would
				// wrongly relabel the previous statement's instruction.
				c.fixDischargedLine(line)
			}
			fs.emit(ABC(OP_ADDI, reg, leftReg, imm+OffsetSC, 0), line)
			fs.emit(ABC(OP_MMBINI, leftReg, imm+OffsetSC, int(TM_ADD), 0), line)
			if tmp {
				fs.freeReg = leftReg
			}
			return
		}
		if imm, ok := smallIntConst(e.Left); ok {
			// imm + a  →  ADDI reg, rightReg, imm; MMBINI rightReg, imm, TM_ADD, k=1
			rightReg, tmp := c.operandToReg(e.Right)
			fs.emit(ABC(OP_ADDI, reg, rightReg, imm+OffsetSC, 0), line)
			fs.emit(ABC(OP_MMBINI, rightReg, imm+OffsetSC, int(TM_ADD), 1), line)
			if tmp {
				fs.freeReg = rightReg
			}
			return
		}
	}

	// Lua 5.5 rewrites  x - <int_literal>  to  x + (-<int_literal>)  and
	// emits OP_ADDI. This is observable on signed-zero floats:
	//   IEEE: (-0.0) + 0.0 = +0.0  but  (-0.0) - 0.0 = -0.0
	// so the rewrite changes the sign of zero in that corner. The
	// metamethod hint still points to TM_SUB and is given the ORIGINAL
	// (unnegated) immediate so __sub(a, n) gets the right operand.
	// smallIntConst's range ([-OffsetSC..OffsetSC]) keeps -imm in-range,
	// so there is no overflow concern (mininteger never reaches here).
	if e.Op == "-" {
		if imm, ok := smallIntConst(e.Right); ok {
			// a - imm  →  ADDI reg, leftReg, -imm; MMBINI leftReg, imm, TM_SUB, 0
			// (MMBINI carries the original imm so the __sub metamethod
			//  receives the user-written argument value.)
			leftReg, tmp := c.operandToReg(e.Left)
			if tmp {
				// See the ADDI case above: skip the line fix when no
				// instruction was emitted for the left operand.
				c.fixDischargedLine(line)
			}
			fs.emit(ABC(OP_ADDI, reg, leftReg, -imm+OffsetSC, 0), line)
			fs.emit(ABC(OP_MMBINI, leftReg, imm+OffsetSC, int(TM_SUB), 0), line)
			if tmp {
				fs.freeReg = leftReg
			}
			return
		}
	}

	// Arithmetic / bitwise. An operand that is a bare in-register local is
	// read in place: its register is used directly as the B/C operand so
	// OP_<op> reads it live at execution time — after the other operand may
	// have mutated it through a shared upvalue — matching reference Lua's
	// exp2anyreg. Compiling it into a temp would both waste a MOVE and
	// snapshot the value before the other operand runs.
	//
	// base snapshots the register top so the epilogue can free every temp
	// reserved here. It must not drop below nActVar: in-place operand
	// registers sit below the temp area and must never be handed back to
	// the allocator (the recurring freeReg bug family).
	base := fs.freeReg
	if base < fs.nActVar {
		base = fs.nActVar
	}
	var leftReg int
	if lr, ok := plainLocalReg(fs, e.Left); ok {
		// In-place local read: no instruction emitted, so skip
		// fixDischargedLine — it would relabel the previous statement's
		// discharge instruction (see the ADDI case above).
		leftReg = lr
	} else if reg < fs.nActVar {
		// reg is a named local: use a fresh temp for the left operand so the
		// right side can still read the local. Example: b = f(a) + b —
		// compiling left into b's register would overwrite it before the
		// right side reads it.
		leftReg = fs.reserveReg()
		c.compileExprToReg(e.Left, leftReg)
		c.fixDischargedLine(line)
		// A call/constructor operand leaves freeReg inflated past its own
		// result slot; reclaim so the right operand lands just above it
		// (same per-operand reset as compileConcat).
		fs.freeReg = leftReg + 1
	} else {
		// reg is a temp or the target of a parent binop: reuse it for the
		// left operand. This keeps left-associative chains like a+b+c+d in
		// O(1) registers instead of O(n).
		leftReg = reg
		c.compileExprToReg(e.Left, leftReg)
		c.fixDischargedLine(line)
		// Reclaim call/constructor inflation (see above). base >= reg+1 here
		// because compileExprToReg's preamble reserved reg before dispatch.
		fs.freeReg = base
	}
	// Right operand: read a plain local in place; otherwise discharge into
	// the current free-register top (rather than reserveReg, which would
	// pre-advance freeReg) so a right operand that is itself a call/table
	// constructor lands directly at rightReg instead of being compiled one
	// slot higher and MOVEd back. The pre-advance would make compileExprToReg
	// see reg < savedFreeReg and take its temp+MOVE branch, leaving a stale
	// register gap below the call target — observable as a spurious
	// "(temporary)" slot in debug.getlocal (see db.lua temp-slot test).
	rightReg, rightInPlace := plainLocalReg(fs, e.Right)
	if !rightInPlace {
		rightReg = fs.freeReg
		c.compileExprToReg(e.Right, rightReg)
	}

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
	// Free every temp reserved for the operands by restoring the entry
	// snapshot. Restoring to leftReg/rightReg instead would be wrong when an
	// operand was a local read in place: those registers sit below the temp
	// area (possibly below live locals) and must stay allocated. For a temp
	// target, base >= reg+1, so the result register stays protected.
	fs.freeReg = base
}

// compileConcat flattens a chain of .. operators (e.g. a .. b .. c) into
// consecutive registers and emits a single OP_CONCAT covering all of them.
func (c *compiler) compileConcat(e *ast.BinopExpr, reg int) {
	fs := c.fs
	// The single folded OP_CONCAT is emitted once the whole chain is parsed, so
	// reference Lua attributes it to the LAST (textually rightmost) '..' operator
	// — which is what a runtime concat error reports. BinopExpr.P is the operator
	// token, so the deepest/rightmost '..' node carries that line. Operand
	// discharges keep their own lines (compileExprToReg below).
	line := concatOpLine(e)

	// Flatten concat chain: a .. b .. c → [a, b, c] in consecutive regs
	exprs := c.flattenConcat(e)

	// Use freeReg as base to avoid clobbering existing locals/for-loop state.
	// If reg < freeReg, we need to work in temporary space and MOVE result back.
	base := fs.freeReg
	needMove := reg < base

	for i, expr := range exprs {
		c.compileExprToReg(expr, base+i)
		// Each operand occupies exactly one register (base+i). A call operand
		// leaves freeReg inflated past its own argument registers, so reset it
		// unconditionally — otherwise the next operand's target (base+i+1) sits
		// below freeReg and takes the temp+MOVE fallback, compounding registers
		// per operand and spuriously overflowing the register file.
		fs.freeReg = base + i + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}

	fs.emit(ABC(OP_CONCAT, base, len(exprs), 0, 0), line)

	if needMove {
		fs.emit(ABC(OP_MOVE, reg, base, 0, 0), line)
	}

	fs.freeReg = base + 1
}

// concatOpLine returns the line of the last (textually rightmost) '..' operator
// in a concat chain. BinopExpr.P is the operator token position, and operators
// are parsed in source order, so the maximum P.Line over all '..' nodes in the
// chain is the last operator — the line reference Lua gives the folded OP_CONCAT.
func concatOpLine(e *ast.BinopExpr) int {
	line := e.P.Line
	var walk func(n ast.Expr)
	walk = func(n ast.Expr) {
		if b, ok := n.(*ast.BinopExpr); ok && b.Op == ".." {
			if b.P.Line > line {
				line = b.P.Line
			}
			walk(b.Left)
			walk(b.Right)
		}
	}
	walk(e)
	return line
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
	// The comparison instruction itself (and the boolean it materializes) is
	// emitted after the right operand is parsed, so reference Lua attributes it
	// to the right operand's end line — which is what a runtime comparison error
	// reports. The left-operand discharge keeps its own line.
	opLine := exprEndLine(e)

	// Emit the comparison itself (immediate form when an operand is a small
	// constant, register form otherwise). It leaves freeReg where it found it.
	base := fs.freeReg
	c.compileComparisonTest(e, false, opLine)

	// comparison + conditional jump → boolean
	jmpFalse := fs.emitJump(opLine) // skip next if comparison fails
	fs.emit(ABC(OP_LOADTRUE, reg, 0, 0, 0), opLine)
	jmpEnd := fs.emitJump(opLine)
	c.patchJump(jmpFalse)
	fs.emit(ABC(OP_LOADFALSE, reg, 0, 0, 0), opLine)
	c.patchJump(jmpEnd)

	fs.freeReg = base
}

// ---------------------------------------------------------------------------
// Logical and/or
// ---------------------------------------------------------------------------

// isComparisonOp returns true if op is a comparison operator.
func isComparisonOp(op string) bool {
	switch op {
	case "==", "~=", "<", "<=", ">", ">=":
		return true
	}
	return false
}

// compileComparisonTest emits the operands and the comparison instruction for
// a comparison expression, WITHOUT the trailing JMP — the caller emits that.
// OP_LT/OP_LE/OP_EQ (and their immediate forms) share OP_TEST's skip
// convention ("pc++ if result ~= k"), so a fused compare is a drop-in
// replacement for a materialized boolean + OP_TEST.
//
// With invertSense=false the caller's JMP fires when the comparison is FALSE
// (the if/while/and sense); invertSense=true flips k so the JMP fires when the
// comparison is TRUE (the or sense). The comparison instruction is attributed
// to opLine; a left operand that had to be discharged into a temp keeps the
// operator's own line, as reference Lua's one-pass compiler does.
// freeReg is restored to its entry value.
func (c *compiler) compileComparisonTest(e *ast.BinopExpr, invertSense bool, opLine int) {
	fs := c.fs

	// Comparisons against a small-integer or string constant use the
	// immediate/constant compare opcodes (EQI/EQK/LTI/LEI/GTI/GEI), like
	// reference luac, saving the constant's LOADI/LOADK.
	if c.tryComparisonImmediate(e, invertSense, opLine) {
		return
	}

	// See compileComparison: read operands in place so an in-register local is
	// compared live. base restores freeReg after the operands.
	base := fs.freeReg
	leftReg, leftTmp := c.operandToReg(e.Left)
	if leftTmp {
		c.fixDischargedLine(e.P.Line)
	}
	rightReg, _ := c.operandToReg(e.Right)

	var op OpCode
	k := 0
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
		leftReg, rightReg = rightReg, leftReg
	case ">=":
		op = OP_LE
		k = 0
		leftReg, rightReg = rightReg, leftReg
	}

	if invertSense {
		k = 1 - k
	}

	fs.emit(ABC(op, leftReg, rightReg, 0, k), opLine)
	fs.freeReg = base
}

// tryComparisonImmediate emits an immediate/constant-operand compare for a
// comparison whose other operand is a small integer constant (EQI/LTI/LEI/
// GTI/GEI) or a string constant (EQK), with compileComparisonTest's k sense.
// It returns false when no immediate form applies, having emitted nothing.
//
// The emitted instruction is semantically identical to LOADI/LOADK + the
// register-form compare: the VM routes the immediate through the same
// equal/lessThan/lessEqual helpers, so comparison metamethods fire with the
// same operands in the same order (reference Lua's luaT_callorderiTM passes
// the immediate in the same position, and flips the operands for GTI/GEI just
// as the register form swaps them for `>`/`>=`).
func (c *compiler) tryComparisonImmediate(e *ast.BinopExpr, invertSense bool, opLine int) bool {
	fs := c.fs

	if e.Op == "==" || e.Op == "~=" {
		k := 0
		if e.Op == "~=" {
			k = 1
		}
		if invertSense {
			k = 1 - k
		}
		// The constant may sit on either side; == and ~= are symmetric, and
		// so are EQI/EQK (they compare R[A] against the immediate/constant).
		regExpr, constExpr := e.Left, e.Right
		if !isImmediateEqConst(constExpr) {
			regExpr, constExpr = constExpr, regExpr
		}
		if imm, ok := smallIntConst(constExpr); ok {
			base := fs.freeReg
			reg, tmp := c.operandToReg(regExpr)
			if tmp && regExpr == e.Left {
				c.fixDischargedLine(e.P.Line)
			}
			fs.emit(ABC(OP_EQI, reg, imm+OffsetSC, 0, k), opLine)
			fs.freeReg = base
			return true
		}
		if s, ok := constExpr.(*ast.StringExpr); ok {
			// The constant index must fit the B field; intern it first so an
			// out-of-range index falls back before any operand is emitted.
			kidx := fs.stringConstant(s.Value)
			if kidx > MaxArgB {
				return false
			}
			base := fs.freeReg
			reg, tmp := c.operandToReg(regExpr)
			if tmp && regExpr == e.Left {
				c.fixDischargedLine(e.P.Line)
			}
			fs.emit(ABC(OP_EQK, reg, kidx, 0, k), opLine)
			fs.freeReg = base
			return true
		}
		return false
	}

	// Order comparison against a small-int operand. A constant on the left
	// flips the mnemonic: K < x  ⇔  x > K, exactly as reference Lua's
	// codeorder rewrites (A < B) to (B > A).
	var op OpCode
	var regExpr ast.Expr
	var imm int
	if v, ok := smallIntConst(e.Right); ok {
		imm, regExpr = v, e.Left
		switch e.Op {
		case "<":
			op = OP_LTI
		case "<=":
			op = OP_LEI
		case ">":
			op = OP_GTI
		case ">=":
			op = OP_GEI
		default:
			return false
		}
	} else if v, ok := smallIntConst(e.Left); ok {
		imm, regExpr = v, e.Right
		switch e.Op {
		case "<":
			op = OP_GTI
		case "<=":
			op = OP_GEI
		case ">":
			op = OP_LTI
		case ">=":
			op = OP_LEI
		default:
			return false
		}
	} else {
		return false
	}

	k := 0
	if invertSense {
		k = 1
	}
	base := fs.freeReg
	reg, tmp := c.operandToReg(regExpr)
	if tmp && regExpr == e.Left {
		c.fixDischargedLine(e.P.Line)
	}
	fs.emit(ABC(op, reg, imm+OffsetSC, 0, k), opLine)
	fs.freeReg = base
	return true
}

// isImmediateEqConst reports whether expr can be the constant operand of an
// EQI/EQK equality compare.
func isImmediateEqConst(expr ast.Expr) bool {
	if _, ok := smallIntConst(expr); ok {
		return true
	}
	_, ok := expr.(*ast.StringExpr)
	return ok
}

// compileComparisonCond compiles a comparison expression as a condition,
// emitting the comparison + JMP. Returns the JMP PC to patch to the false
// path. The comparison is compiled with the given k sense: k=0 means
// "skip next (JMP) if comparison is TRUE" (used by and),
// k=1 means "skip next (JMP) if comparison is FALSE" (used by or).
func (c *compiler) compileComparisonCond(e *ast.BinopExpr, invertSense bool) int {
	line := e.P.Line
	c.compileComparisonTest(e, invertSense, line)
	return c.fs.emitJump(line) // JMP to false/true path
}

// compileAnd compiles "a and b" — short-circuits to a if a is falsy.
//
// When the left operand is a comparison, the comparison's conditional jump
// is fused with the and short-circuit, avoiding boolean materialization.
// This matches reference Lua 5.4's codegen and is critical for keeping
// instruction counts compatible with debug count hooks.
func (c *compiler) compileAnd(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	// Optimization: if left is a comparison, fuse with and short-circuit.
	// Instead of: materialize boolean → TESTSET → JMP,
	// emit: comparison → JMP-to-false (2 instructions, not 7).
	// Uses LOADFALSE (not LFALSESKIP) on the false path to avoid
	// skipping the wrong instruction when nested inside or/if.
	if cmp, ok := e.Left.(*ast.BinopExpr); ok && isComparisonOp(cmp.Op) {
		// comparison skips JMP if TRUE (invertSense=false → k=0 for <,<=,==)
		// so JMP fires when comparison is FALSE → short-circuit
		falseJmp := c.compileComparisonCond(cmp, false)

		// Compile right operand normally.
		c.compileExprToReg(e.Right, reg)

		endJmp := fs.emitJump(line) // jump over false path
		c.patchJump(falseJmp)
		fs.emit(ABC(OP_LOADFALSE, reg, 0, 0, 0), line)
		c.patchJump(endJmp)
		return
	}

	if reg < fs.nActVar {
		// reg is a live local (assignment target): evaluate the left operand
		// into a fresh temp so it can't clobber the local before the right
		// operand reads it (e.g. n = (n==0) or stack(n-1)). TESTSET copies the
		// kept value into reg on the short-circuit path.
		tmp := fs.reserveReg()
		c.compileExprToReg(e.Left, tmp)
		fs.emit(ABC(OP_TESTSET, reg, tmp, 0, 0), line) // skip if falsy, keep value
		jmp := fs.emitJump(line)
		c.compileExprToReg(e.Right, reg)
		c.patchJump(jmp)
		fs.freeReg = tmp
		return
	}
	// reg is a temporary: evaluate the left operand into it and test in place,
	// so a chain (a and b and c ...) reuses one register instead of a fresh
	// temp per level (which overflowed the 255-register limit at ~255 operands
	// on programs reference Lua compiles). TEST reg,k=0 skips the short-circuit
	// JMP when reg is truthy -> evaluate the right operand into reg; otherwise
	// the JMP keeps reg = the (falsy) left value.
	c.compileExprToReg(e.Left, reg)
	// A call left operand leaves freeReg inflated past reg; reset it so the
	// right operand compiles in place at reg instead of taking the temp+MOVE
	// fallback (which compounds registers per level and overflows long chains).
	fs.freeReg = reg + 1
	fs.emit(ABC(OP_TEST, reg, 0, 0, 0), line)
	jmp := fs.emitJump(line)
	c.compileExprToReg(e.Right, reg)
	c.patchJump(jmp)
}

// compileOr compiles "a or b" — short-circuits to a if a is truthy.
//
// When the left operand is a comparison, the comparison's conditional jump
// is fused with the or short-circuit (inverted sense).
func (c *compiler) compileOr(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	// Optimization: if left is a comparison, fuse with or short-circuit.
	if cmp, ok := e.Left.(*ast.BinopExpr); ok && isComparisonOp(cmp.Op) {
		// For or: skip JMP if comparison is FALSE (invertSense=true),
		// because or short-circuits when left is truthy.
		trueJmp := c.compileComparisonCond(cmp, true)

		// Compile right operand
		c.compileExprToReg(e.Right, reg)

		endJmp := fs.emitJump(line) // jump over true path
		c.patchJump(trueJmp)
		fs.emit(ABC(OP_LOADTRUE, reg, 0, 0, 0), line)
		c.patchJump(endJmp) // both paths converge here
		return
	}

	if reg < fs.nActVar {
		// reg is a live local: use a fresh temp for the left operand — see
		// compileAnd.
		tmp := fs.reserveReg()
		c.compileExprToReg(e.Left, tmp)
		fs.emit(ABC(OP_TESTSET, reg, tmp, 0, 1), line) // skip if truthy, keep value
		jmp := fs.emitJump(line)
		c.compileExprToReg(e.Right, reg)
		c.patchJump(jmp)
		fs.freeReg = tmp
		return
	}
	// reg is a temporary: test in place so an or-chain reuses one register.
	// TEST reg,k=1 skips the short-circuit JMP when reg is falsy -> evaluate the
	// right operand into reg; otherwise the JMP keeps reg = the (truthy) left.
	c.compileExprToReg(e.Left, reg)
	// See compileAnd: reset inflated freeReg from a call left operand.
	fs.freeReg = reg + 1
	fs.emit(ABC(OP_TEST, reg, 0, 0, 1), line)
	jmp := fs.emitJump(line)
	c.compileExprToReg(e.Right, reg)
	c.patchJump(jmp)
}

// ---------------------------------------------------------------------------
// Unary operations
// ---------------------------------------------------------------------------

// compileUnop compiles a unary operator: -x (UNM), not x (NOT), #t (LEN), ~n (BNOT).
func (c *compiler) compileUnop(e *ast.UnopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	// Read a plain in-register local operand in place (reference Lua's
	// exp2anyreg) instead of snapshotting it into reg with a MOVE.
	opReg, inPlace := plainLocalReg(fs, e.Operand)
	freeTo := -1
	if !inPlace {
		if reg < fs.regTop() {
			// reg belongs to a live variable (or a still-live condition temp
			// below the locals top): evaluating the operand there would make
			// the target visibly hold the operand while a metamethod runs, and
			// would blame the target in the error message when there is none.
			// Reference codeunexpval evaluates into a temp and leaves the
			// result relocatable, so the store happens last.
			opReg = fs.reserveReg()
			freeTo = opReg
		} else {
			// reg is a temp: evaluate in place, as reference's exp2anyreg does
			// when the operand already owns the top register.
			opReg = reg
		}
		c.compileExprToReg(e.Operand, opReg)
	}

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
	if freeTo >= 0 {
		fs.freeReg = freeTo
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

	// base is always a fresh call-frame register: callers that target a live
	// local wrap the call in a temp+MOVE, and the CALL overwrites base upward
	// with its results, so nothing above base is live here. Normalize freeReg
	// down to base before compiling the function expression so that a chained
	// call (f(x)(y)) reuses base for its single result instead of taking the
	// temp+MOVE fallback at base+1 — the latter grew the register frame by one
	// per chain level and overflowed the register file (Go index panic).
	if base < fs.freeReg {
		fs.freeReg = base
	}

	// Function goes into base.
	c.compileExprToReg(e.Func, base)

	// Reset freeReg after compiling the function expression. If e.Func is a
	// chained call (e.g. f(x)(y)), compileExprToReg inflates freeReg for the
	// inner call's arguments; reset so arguments compile into base+1, base+2...
	fs.freeReg = base + 1
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	if fs.freeReg > fs.c.limits.MaxRegs {
		fs.limitError("registers", fs.c.limits.MaxRegs, line, "")
	}

	// Arguments into base+1, base+2, ...
	nArgs := len(e.Args)
	for i, arg := range e.Args {
		if i == nArgs-1 && isMultiRet(arg) {
			c.compileExprMultiRet(arg, 0)                     // open call
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

	// base is always a fresh call-frame register (callers that target a live
	// local wrap the call in a temp+MOVE). Normalize freeReg down to base and
	// compile the receiver in place: SELF reads R[B] before writing, so with
	// B == A it copies the receiver to base+1 while loading the method into
	// base. A chain o:m():n()... then reuses one register pair per link
	// instead of holding two per level, which overflowed the register file
	// at ~128 links on chains reference Lua runs in constant registers.
	if base < fs.freeReg {
		fs.freeReg = base
	}
	c.compileExprToReg(e.Object, base)
	methodK := fs.stringConstant(e.Method)
	fs.emitSelf(base, base, methodK, line)

	// Function at base, self at base+1; arguments start at base+2. Reset
	// unconditionally: compiling the receiver may have inflated freeReg
	// (e.g. inner call arguments), and compileExprMultiRet for the last
	// argument must start at base+2, not at the inflated value.
	fs.freeReg = base + 2
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	if fs.freeReg > fs.c.limits.MaxRegs {
		fs.limitError("registers", fs.c.limits.MaxRegs, line, "")
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
	fs.freeReg = base + 1
}

// ---------------------------------------------------------------------------
// Table constructors
// ---------------------------------------------------------------------------

// compileTableConstructor compiles {1, 2, key="val"} — emits NEWTABLE,
// then SETLIST for array entries and SETFIELD/SETTABLE for hash entries.
func (c *compiler) compileTableConstructor(e *ast.TableConstructor, reg int) {
	fs := c.fs
	line := e.P.Line

	// Ensure freeReg is past the table register so that reserveReg() for
	// field values does not return reg itself, which would clobber the table.
	if fs.freeReg <= reg {
		fs.freeReg = reg + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
		fs.checkRegLimit()
	}

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
	if vC > MaxArgVC {
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

	// Find the last array-style field to check if it's multi-return.
	// Multi-return expansion only happens if the call/vararg is the very
	// last field in the constructor (including hash fields).
	lastArrayIdx := -1
	lastFieldIdx := len(e.Fields) - 1
	for i, f := range e.Fields {
		if f.Key == nil {
			lastArrayIdx = i
		}
	}

	// Compute the dynamic SETLIST flush threshold based on available registers.
	// This follows the Lua 5.5 algorithm (lparser.c maxtostore) which replaces
	// the fixed 50 (LFIELDS_PER_FLUSH) to allow deeper constructor nesting.
	flushThreshold := fs.maxToStore()

	// Fill table fields
	arrIdx := 0
	pendingList := 0 // number of pending SETLIST items
	for i, f := range e.Fields {
		if f.Key == nil {
			// Array-style field
			arrIdx++
			pendingList++
			arrReg := reg + pendingList
			if fs.freeReg <= arrReg {
				// Let the value compile directly into the next pending slot;
				// compileExprToReg reserves it with the register-limit check.
				fs.freeReg = arrReg
			}

			// Check if this is the last array element, the last field overall,
			// and it's a multi-return expression. Multi-return only expands
			// when the call/vararg is the very last field in the constructor.
			if i == lastArrayIdx && i == lastFieldIdx && isMultiRet(f.Value) {
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
			// Keep exactly the pending slots reserved (drop stale value temps)
			fs.freeReg = arrReg + 1

			// Flush when pending items reach the dynamic threshold
			if pendingList >= flushThreshold {
				c.emitSetList(reg, pendingList, arrIdx-pendingList+1, line)
				pendingList = 0
				fs.freeReg = reg + 1
			}
		} else {
			// Hash-style field
			switch key := f.Key.(type) {
			case *ast.StringExpr:
				startReg := fs.freeReg
				valC, valK := c.storeValueOperand(f.Value)
				kIdx := fs.stringConstant(key.Value)
				fs.emitSetField(reg, kIdx, valC, valK, line)
				fs.freeReg = startReg
			case *ast.NumberExpr:
				if key.Value >= 0 && key.Value <= int64(MaxArgC) {
					startReg := fs.freeReg
					valC, valK := c.storeValueOperand(f.Value)
					fs.emit(ABC(OP_SETI, reg, int(key.Value), valC, valK), line)
					fs.freeReg = startReg
				} else {
					keyReg := fs.reserveReg()
					c.compileExprToReg(f.Key, keyReg)
					valC, valK := c.storeValueOperand(f.Value)
					fs.emit(ABC(OP_SETTABLE, reg, keyReg, valC, valK), line)
					fs.freeReg = keyReg
				}
			default:
				keyReg := fs.reserveReg()
				c.compileExprToReg(f.Key, keyReg)
				valC, valK := c.storeValueOperand(f.Value)
				fs.emit(ABC(OP_SETTABLE, reg, keyReg, valC, valK), line)
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
	if vC > MaxArgVC {
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
	// When the table is a local variable, use its register directly.
	tableReg := -1
	if name, ok := unparen(e.Table).(*ast.NameExpr); ok {
		if localReg, found := fs.lookupLocal(name.Name); found {
			tableReg = localReg
		}
	}
	needFree := false
	if tableReg < 0 {
		if reg < fs.nActVar {
			// reg is a live local (assignment target): use a fresh temp so the
			// table sub-expression can't clobber the local prematurely.
			tableReg = fs.reserveReg()
			needFree = true
		} else {
			// reg is a temporary: index it in place so a chain (t.a.b.c...)
			// reuses one register instead of one per level (which overflowed
			// the 255-register limit at depth ~255 on programs reference Lua
			// compiles fine). Inner chain levels then also see a temp and reuse.
			tableReg = reg
		}
		c.compileExprToReg(e.Table, tableReg)
	}
	fieldK := fs.stringConstant(e.Field)
	fs.emitGetField(reg, tableReg, fieldK, e.P.Line)
	if needFree {
		fs.freeReg = tableReg
	}
}

// compileIndexExpr compiles t[key] into GETI (constant int 0-255) or GETTABLE.
func (c *compiler) compileIndexExpr(e *ast.IndexExpr, reg int) {
	fs := c.fs
	// When the table is a local variable, use its register directly to
	// avoid emitting an unnecessary MOVE. This matches Lua 5.4's behavior
	// where VINDEXED expressions reference the table register directly.
	tableReg := -1
	if name, ok := unparen(e.Table).(*ast.NameExpr); ok {
		if localReg, found := fs.lookupLocal(name.Name); found {
			tableReg = localReg
		}
	}
	needFree := false
	if tableReg < 0 {
		if reg < fs.nActVar {
			// reg is a live local: use a fresh temp so neither the table
			// sub-expression nor a non-constant key clobbers the local before
			// it is read (e.g. x = a[x][x]).
			tableReg = fs.reserveReg()
			needFree = true
		} else {
			// reg is a temporary: index it in place so a chain (t[a][b]...)
			// reuses one register — see compileFieldExpr.
			tableReg = reg
		}
		c.compileExprToReg(e.Table, tableReg)
	}
	if n, ok := e.Key.(*ast.NumberExpr); ok && n.Value >= 0 && n.Value <= int64(MaxArgC) {
		fs.emit(ABC(OP_GETI, reg, tableReg, int(n.Value), 0), e.P.Line)
		if needFree {
			fs.freeReg = tableReg
		}
	} else {
		keyReg := fs.reserveReg()
		c.compileExprToReg(e.Key, keyReg)
		fs.emit(ABC(OP_GETTABLE, reg, tableReg, keyReg, 0), e.P.Line)
		if needFree {
			fs.freeReg = tableReg // frees tableReg and keyReg (keyReg > tableReg)
		} else {
			fs.freeReg = keyReg // reg is caller-owned; free only the key temp
		}
	}
}

// powWithSubnormalFix computes x^y with a rescaling fix for positive
// subnormal (denormal) bases, matching libm's pow. See vm/vm_pow.go for
// the full rationale. Duplicated here because the compiler package must
// not import vm.
func powWithSubnormalFix(x, y float64) float64 {
	return libmNaNSignFix(x, y, powWithSubnormalFixImpl(x, y))
}

func powWithSubnormalFixImpl(x, y float64) float64 {
	if x == 0 || math.IsNaN(x) || math.IsInf(x, 0) || math.IsNaN(y) {
		return math.Pow(x, y)
	}
	const smallestNormal = 2.2250738585072014e-308 // 2^-1022
	if math.Abs(x) >= smallestNormal {
		return math.Pow(x, y)
	}
	if x < 0 {
		return math.Pow(x, y)
	}
	mantBits := math.Float64bits(x) & ((1 << 52) - 1)
	m := float64(mantBits)
	return math.Pow(m, y) * math.Exp2(-1074.0*y)
}

// libmNaNSignFix mirrors vm.libmNaNSignFix. See vm/vm_pow.go for
// rationale. Duplicated because the compiler package cannot import vm.
func libmNaNSignFix(x, y, r float64) float64 {
	if !math.IsNaN(r) {
		return r
	}
	xIsNaN := math.IsNaN(x)
	yIsNaN := math.IsNaN(y)
	switch {
	case xIsNaN && yIsNaN:
		return math.Copysign(r, libmSignOf(x))
	case xIsNaN:
		if y == 1 || y == -1 {
			return math.Copysign(r, +1)
		}
		if libmIsOddInt(y) {
			return math.Copysign(r, -libmSignOf(x))
		}
		return math.Copysign(r, libmSignOf(x))
	case yIsNaN:
		return math.Copysign(r, libmSignOf(y))
	default:
		return math.Copysign(r, -1)
	}
}

func libmSignOf(f float64) float64 {
	if math.Signbit(f) {
		return -1
	}
	return 1
}

func libmIsOddInt(y float64) bool {
	if math.IsNaN(y) || math.IsInf(y, 0) {
		return false
	}
	if math.Abs(y) >= (1 << 53) {
		return false
	}
	yi, yf := math.Modf(y)
	return yf == 0 && int64(yi)&1 == 1
}
