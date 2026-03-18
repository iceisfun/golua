package compiler

import (
	"github.com/iceisfun/golua/ast"
)

// preRegisterUpvalues walks an assignment target expression and pre-registers
// any upvalue references in left-to-right order. This ensures that upvalue
// indices match Lua 5.4's source order, where LHS targets are processed
// before RHS values during compilation.
func (c *compiler) preRegisterUpvalues(fs *funcState, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.NameExpr:
		if _, isLocal := fs.lookupLocal(e.Name); !isLocal {
			c.resolveUpvalue(fs, e.Name)
		}
	case *ast.IndexExpr:
		c.preRegisterUpvalues(fs, e.Table)
		c.preRegisterUpvalues(fs, e.Key)
	case *ast.FieldExpr:
		c.preRegisterUpvalues(fs, e.Table)
	}
}

// ---------------------------------------------------------------------------
// Chunk compilation
// ---------------------------------------------------------------------------

// compileChunk creates the top-level Proto for a Lua source file. The chunk
// is compiled as a vararg function with _ENV as its first upvalue.
func (c *compiler) compileChunk(source string, block *ast.Block) *Proto {
	fs := c.newFuncState(source, nil)
	fs.maxReg = 2 // minimum

	// Top-level chunk is a vararg function
	fs.proto.IsVarArg = true
	fs.proto.NumParams = 0
	fs.proto.LineDef = 0
	fs.proto.LastLine = 0

	// _ENV is upvalue[0] for the top-level chunk
	fs.addUpvalue("_ENV", true, 0)

	fs.enterScope(false)

	// Emit VARARGPREP
	line := 0
	if block != nil && len(block.Stmts) > 0 {
		line = block.Start.Line
	}
	fs.emit(ABC(OP_VARARGPREP, 0, 0, 0, 0), line)

	c.compileBlock(block)

	// Emit final return
	lastLine := line
	if block != nil && len(block.Stmts) > 0 {
		lastLine = block.Stmts[len(block.Stmts)-1].Pos().Line
	}
	fs.emit(ABC(OP_RETURN0, 0, 0, 0, 0), lastLine)
	// LastLine stays 0 for the main chunk (set at proto init).
	// Lua 5.4 always reports lastlinedefined=0 for the top-level function.

	c.leaveScope(lastLine)

	return c.closeFuncState()
}

// ---------------------------------------------------------------------------
// Block and statement compilation
// ---------------------------------------------------------------------------

// compileBlock compiles all statements in a block, releasing temporaries after each.
func (c *compiler) compileBlock(block *ast.Block) {
	c.compileBlockWith(block, true, 0)
}

// compileBlockWith compiles a block. When labelEndOpt is true, labels at
// the end of the block are treated as if preceding locals are out of scope
// (Lua 5.4 §3.3.4). This must be false for repeat-until bodies, where the
// until condition can still reference body locals.
//
// blockAfterLine is the line of whatever follows this block (e.g., the
// until keyword for repeat blocks, or the end keyword's line). When > 0,
// it is used as the fallback error line for labels at the end of the block.
func (c *compiler) compileBlockWith(block *ast.Block, labelEndOpt bool, blockAfterLine int) {
	if block == nil {
		return
	}
	stmts := block.Stmts
	for i := 0; i < len(stmts); i++ {
		stmt := stmts[i]
		// When compiling a label, tell it whether it's at the end of
		// the block (followed only by other labels). Lua 5.4 treats
		// such labels as if locals declared before them are already
		// out of scope, allowing goto to jump over those locals.
		if _, ok := stmt.(*ast.LabelStmt); ok {
			// Collect consecutive label statements. Lua 5.4's labelstat()
			// recursively processes adjacent labels before registering the
			// current one, effectively registering them in reverse order.
			// This affects duplicate detection: for "::L::\n::L::", the
			// second label is registered first, so the error says "already
			// defined on line 2" (the second label's line).
			runEnd := i + 1
			for runEnd < len(stmts) {
				if _, isLabel := stmts[runEnd].(*ast.LabelStmt); !isLabel {
					break
				}
				runEnd++
			}

			// Compute afterLine for the entire label run.
			afterLine := blockAfterLine
			if afterLine == 0 {
				afterLine = c.endLine
			}
			if runEnd < len(stmts) {
				afterLine = stmts[runEnd].Pos().Line
			}

			// Process labels in reverse order to match lua5.4's recursive behavior.
			for j := runEnd - 1; j >= i; j-- {
				lbl := stmts[j].(*ast.LabelStmt)
				atEnd := labelEndOpt && labelAtBlockEnd(stmts, j)
				c.compileLabelStmt(lbl, atEnd, afterLine)
			}

			// Skip past the label run (loop will increment i).
			i = runEnd - 1
			continue
		} else if ls, ok := stmt.(*ast.LocalStmt); ok {
			// Pass next-statement info for accurate "too many locals" error messages.
			nextLine, nextNear := c.nextStmtInfo(stmts, i, blockAfterLine)
			c.compileLocalStmtWithNext(ls, nextLine, nextNear)
		} else {
			c.compileStmt(stmt)
		}
		// After each statement, release temporary registers while
		// preserving all registers occupied by active locals.
		// We use regTop() instead of nActVar because locals may not
		// occupy contiguous registers starting from R(0) — condition
		// temporaries (e.g., while/for loop conditions) can create gaps.
		c.fs.freeReg = c.fs.regTop()
	}
}

// nextStmtInfo returns the line and near-token for the statement following
// stmts[idx], matching Lua 5.4's lookahead behavior for error messages.
func (c *compiler) nextStmtInfo(stmts []ast.Stmt, idx int, blockAfterLine int) (int, string) {
	if idx+1 < len(stmts) {
		next := stmts[idx+1]
		return next.Pos().Line, stmtNearToken(next)
	}
	line := blockAfterLine
	if line == 0 {
		line = c.endLine
	}
	near := "<eof>"
	if blockAfterLine > 0 {
		near = "end"
	}
	return line, near
}

// stmtNearToken returns the leading keyword/token for a statement.
func stmtNearToken(s ast.Stmt) string {
	switch s.(type) {
	case *ast.LocalStmt, *ast.LocalFuncStmt:
		return "local"
	case *ast.IfStmt:
		return "if"
	case *ast.WhileStmt:
		return "while"
	case *ast.RepeatStmt:
		return "repeat"
	case *ast.ForNumStmt, *ast.ForInStmt:
		return "for"
	case *ast.DoStmt:
		return "do"
	case *ast.ReturnStmt:
		return "return"
	case *ast.BreakStmt:
		return "break"
	case *ast.GotoStmt:
		return "goto"
	case *ast.LabelStmt:
		return "::"
	case *ast.FuncStmt:
		return "function"
	case *ast.EmptyStmt:
		return ";"
	case *ast.GlobalStmt, *ast.GlobalFuncStmt:
		return "global"
	case *ast.ExprStmt:
		return exprLeadToken(s.(*ast.ExprStmt).Expr)
	case *ast.AssignStmt:
		return exprLeadToken(s.(*ast.AssignStmt).Targets[0])
	default:
		return "<eof>"
	}
}

// exprLeadToken returns the leading token text for an expression.
func exprLeadToken(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.NameExpr:
		return x.Name
	case *ast.ParenExpr:
		return "("
	default:
		return "<eof>"
	}
}

// labelAtBlockEnd reports whether the statement at stmts[idx] is a label
// followed only by other labels or semicolons before the block ends.
// Lua 5.4 treats such labels as if locals declared before them are
// already out of scope.
func labelAtBlockEnd(stmts []ast.Stmt, idx int) bool {
	for j := idx + 1; j < len(stmts); j++ {
		switch stmts[j].(type) {
		case *ast.LabelStmt, *ast.EmptyStmt:
			// no-op statements — continue checking
		default:
			return false
		}
	}
	return true
}

// compileStmt dispatches a single statement to its specific compiler method.
func (c *compiler) compileStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.LocalStmt:
		c.compileLocalStmt(s)
	case *ast.AssignStmt:
		c.compileAssignStmt(s)
	case *ast.ExprStmt:
		c.compileExprStmt(s)
	case *ast.ReturnStmt:
		c.compileReturnStmt(s)
	case *ast.IfStmt:
		c.compileIfStmt(s)
	case *ast.WhileStmt:
		c.compileWhileStmt(s)
	case *ast.RepeatStmt:
		c.compileRepeatStmt(s)
	case *ast.DoStmt:
		c.compileDoStmt(s)
	case *ast.ForNumStmt:
		c.compileForNumStmt(s)
	case *ast.ForInStmt:
		c.compileForInStmt(s)
	case *ast.BreakStmt:
		c.compileBreakStmt(s)
	case *ast.GotoStmt:
		c.compileGotoStmt(s)
	case *ast.LabelStmt:
		c.compileLabelStmt(s, false, c.endLine)
	case *ast.FuncStmt:
		c.compileFuncStmt(s)
	case *ast.LocalFuncStmt:
		c.compileLocalFuncStmt(s)
	case *ast.GlobalStmt:
		c.compileGlobalStmt(s)
	case *ast.GlobalFuncStmt:
		c.compileGlobalFuncStmt(s)
	case *ast.EmptyStmt:
		// nothing
	default:
		c.error(stmt, "unhandled statement type %T", stmt)
	}
}

// ---------------------------------------------------------------------------
// Local declarations
// ---------------------------------------------------------------------------

// compileLocalStmt compiles "local x, y = a, b" — evaluates RHS into
// consecutive registers, then declares the local variables.
func (c *compiler) compileLocalStmt(s *ast.LocalStmt) {
	c.compileLocalStmtWithNext(s, 0, "")
}

// compileLocalStmtWithNext compiles a local statement with info about the
// next statement, used for "too many locals" error messages to match
// Lua 5.4's lookahead-based line/near reporting.
func (c *compiler) compileLocalStmtWithNext(s *ast.LocalStmt, nextLine int, nextNear string) {
	fs := c.fs
	line := s.P.Line
	nNames := len(s.Names)
	nValues := len(s.Values)

	// Base register — locals will occupy base..base+nNames-1
	base := fs.freeReg

	// Compile all RHS values into base, base+1, ...
	lastIsMultiRet := false
	if nValues > 0 {
		for i := 0; i < nValues; i++ {
			if i == nValues-1 && i < nNames-1 && isMultiRet(s.Values[i]) {
				// Last expression is multi-return, needs to fill remaining slots
				c.compileExprMultiRet(s.Values[i], nNames-i)
				lastIsMultiRet = true
			} else if i < nNames {
				c.compileExprToReg(s.Values[i], base+i)
				// Reset freeReg to only what we've committed
				fs.freeReg = base + i + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			} else {
				// More values than names — evaluate for side effects into temp
				tmp := fs.freeReg
				c.compileExprToReg(s.Values[i], tmp)
				fs.freeReg = base + nNames // discard temp
			}
		}

		// Fill missing values with nil (but not if last expr was multi-return)
		if nValues < nNames && !lastIsMultiRet {
			fs.emit(ABC(OP_LOADNIL, base+nValues, nNames-nValues-1, 0, 0), line)
		}
	} else {
		// No values — fill all with nil
		fs.emit(ABC(OP_LOADNIL, base, nNames-1, 0, 0), line)
	}

	// Register all local variables occupying base..base+nNames-1
	// Choose near token and line to match Lua 5.4:
	//   - If any variable has an attribute, near '<' (attribute opener), current line
	//   - If there are values (=), near '=', current line
	//   - Otherwise, use the next statement's line and near token (lookahead)
	nearToken := ""
	errLine := line
	hasAttrib := false
	for _, a := range s.Attribs {
		if a != "" {
			hasAttrib = true
			break
		}
	}
	if hasAttrib {
		nearToken = "<"
	} else if nValues > 0 {
		nearToken = "="
	} else {
		// No values, no attributes — use next token info (Lua 5.4 lookahead)
		if nextNear != "" {
			nearToken = nextNear
			errLine = nextLine
		} else {
			nearToken = "<eof>"
		}
	}
	fs.checkVarLimitAt(nNames, errLine, nearToken)

	fs.freeReg = base + nNames
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	fs.checkRegLimit()
	baseIdx := len(fs.locals)
	for i, name := range s.Names {
		attrib := ""
		if i < len(s.Attribs) {
			attrib = s.Attribs[i]
		}
		fs.locals = append(fs.locals, localVar{
			name:    name.Name,
			reg:     base + i,
			startPC: -1,
			attrib:  attrib,
		})
		fs.nActVar++
	}

	// Activate all locals at the current PC
	for i := 0; i < nNames; i++ {
		fs.locals[baseIdx+i].startPC = fs.pc()
	}

	// Emit OP_TBC for <close> variables
	for i := 0; i < nNames; i++ {
		if fs.locals[baseIdx+i].attrib == "close" {
			fs.emit(ABC(OP_TBC, base+i, 0, 0, 0), line)
		}
	}
}

// ---------------------------------------------------------------------------
// Assignment
// ---------------------------------------------------------------------------

// compileAssignStmt compiles "a, b = x, y". For single assignments it
// optimizes directly; for multi-assign it evaluates all RHS into temps first.
func (c *compiler) compileAssignStmt(s *ast.AssignStmt) {
	fs := c.fs
	line := s.P.Line
	nTargets := len(s.Targets)
	nValues := len(s.Values)

	// For single assignment with single value, optimize
	if nTargets == 1 && nValues == 1 {
		c.compileSingleAssign(s.Targets[0], s.Values[0], line)
		return
	}

	// General case: evaluate all values into temp regs, then assign.
	//
	// For correctness, LHS table/key sub-expressions (e.g. a[i]) must be
	// evaluated before any assignment occurs, because a later assignment
	// might overwrite a variable used in an earlier LHS index expression.
	// Example: i, a[i], a = j, i, i  — a[i] must use the original a and i.

	// Phase 0: Pre-register upvalues referenced by LHS targets in
	// left-to-right order. This ensures upvalue indices match Lua 5.4's
	// source order, where targets are processed before RHS values.
	for i := 0; i < nTargets; i++ {
		c.preRegisterUpvalues(fs, s.Targets[i])
	}

	// Phase 1: Pre-evaluate LHS indexed targets into temp registers.
	type precomputedTarget struct {
		tableReg int // temp reg holding table reference
		keyReg   int // temp reg holding key (-1 for field/intKey targets)
		fieldK   int // constant index for field targets (-1 otherwise)
		intKey   int // constant integer key for SETI (-1 if not applicable)
	}
	precomputed := make([]precomputedTarget, nTargets)
	tempBase := fs.freeReg

	for i := 0; i < nTargets; i++ {
		switch t := s.Targets[i].(type) {
		case *ast.IndexExpr:
			tReg := fs.reserveReg()
			c.compileExprToReg(t.Table, tReg)
			if n, ok := t.Key.(*ast.NumberExpr); ok && n.Value >= 0 && n.Value <= int64(MaxArgC) {
				precomputed[i] = precomputedTarget{tableReg: tReg, keyReg: -1, fieldK: -1, intKey: int(n.Value)}
			} else {
				kReg := fs.reserveReg()
				c.compileExprToReg(t.Key, kReg)
				precomputed[i] = precomputedTarget{tableReg: tReg, keyReg: kReg, fieldK: -1, intKey: -1}
			}
		case *ast.FieldExpr:
			tReg := fs.reserveReg()
			c.compileExprToReg(t.Table, tReg)
			fK := fs.stringConstant(t.Field)
			precomputed[i] = precomputedTarget{tableReg: tReg, keyReg: -1, fieldK: fK, intKey: -1}
		default:
			precomputed[i] = precomputedTarget{tableReg: -1, keyReg: -1, fieldK: -1, intKey: -1}
		}
	}

	// Phase 2: Evaluate all RHS values into temp registers.
	valBase := fs.freeReg
	lastIsMultiRet := false

	for i := 0; i < nValues; i++ {
		if i == nValues-1 && nValues < nTargets {
			// Last expression, might be multi-return
			if isMultiRet(s.Values[i]) {
				lastIsMultiRet = true
				c.compileExprMultiRet(s.Values[i], nTargets-i)
			} else {
				reg := valBase + i
				c.compileExprToReg(s.Values[i], reg)
				// Reset freeReg to reclaim temporaries used internally
				fs.freeReg = reg + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			}
		} else {
			reg := valBase + i
			c.compileExprToReg(s.Values[i], reg)
			// Reset freeReg to reclaim temporaries used internally
			// (e.g., nested function call arguments). Without this,
			// compileExprMultiRet for the next value would start at
			// an inflated freeReg, leaving stale values in the gap.
			fs.freeReg = reg + 1
			if fs.freeReg > fs.maxReg {
				fs.maxReg = fs.freeReg
			}
		}
	}

	// Fill missing values with nil (but not if last expr was multi-return)
	if nValues < nTargets && !lastIsMultiRet {
		for i := nValues; i < nTargets; i++ {
			reg := valBase + i
			fs.emit(ABC(OP_LOADNIL, reg, 0, 0, 0), line)
			if reg >= fs.freeReg {
				fs.freeReg = reg + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			}
		}
	}

	// Phase 3: Assign from temp value registers to targets using precomputed
	// table/key registers for indexed targets.
	// Lua 5.4 assigns right-to-left so that in `t[1], t[1] = "a", "b"`,
	// t[1] ends up as "a" (the leftmost assignment wins).
	for i := nTargets - 1; i >= 0; i-- {
		pc := precomputed[i]
		if pc.tableReg >= 0 {
			// Pre-evaluated indexed/field target
			if pc.intKey >= 0 {
				fs.emit(ABC(OP_SETI, pc.tableReg, pc.intKey, valBase+i, 0), line)
			} else if pc.keyReg >= 0 {
				fs.emit(ABC(OP_SETTABLE, pc.tableReg, pc.keyReg, valBase+i, 0), line)
			} else if pc.fieldK >= 0 {
				fs.emitSetField(pc.tableReg, pc.fieldK, valBase+i, line)
			}
		} else {
			c.assignToTarget(s.Targets[i], valBase+i, line)
		}
	}

	fs.freeReg = tempBase
}

// compileSingleAssign handles the common case of "x = expr" — assigns to
// a local, upvalue, global, table field, or table index.
func (c *compiler) compileSingleAssign(target ast.Expr, value ast.Expr, line int) {
	fs := c.fs

	switch t := target.(type) {
	case *ast.NameExpr:
		// Local?
		if reg, ok := fs.lookupLocal(t.Name); ok {
			if fs.isConst(t.Name) {
				c.error(target, "attempt to assign to const variable '%s'", t.Name)
				return
			}
			// compileExprToReg handles clobber protection for function/method
			// calls via savedFreeReg — no special-casing needed here.
			c.compileExprToReg(value, reg)
			return
		}
		// Upvalue?
		if idx, ok := c.resolveUpvalue(fs, t.Name); ok {
			_ = idx
			if c.isConstUpvalue(fs, t.Name) {
				c.error(target, "attempt to assign to const variable '%s'", t.Name)
				return
			}
			tempReg := fs.reserveReg()
			c.compileExprToReg(value, tempReg)
			fs.emit(ABC(OP_SETUPVAL, tempReg, idx, 0, 0), line)
			fs.freeReg = tempReg
			return
		}
		// Local _ENV: _ENV[name] via SETFIELD on local
		if envReg, ok := fs.lookupLocal("_ENV"); ok {
			nameK := fs.stringConstant(t.Name)
			tempReg := fs.reserveReg()
			c.compileExprToReg(value, tempReg)
			fs.emitSetField(envReg, nameK, tempReg, line)
			fs.freeReg = tempReg
			return
		}
		// Global: _ENV[name]
		c.compileSetGlobal(t.Name, value, line)

	case *ast.FieldExpr:
		tableReg := fs.reserveReg()
		c.compileExprToReg(t.Table, tableReg)
		valReg := fs.reserveReg()
		c.compileExprToReg(value, valReg)
		fieldK := fs.stringConstant(t.Field)
		fs.emitSetField(tableReg, fieldK, valReg, line)
		fs.freeReg = tableReg

	case *ast.IndexExpr:
		tableReg := fs.reserveReg()
		c.compileExprToReg(t.Table, tableReg)
		if n, ok := t.Key.(*ast.NumberExpr); ok && n.Value >= 0 && n.Value <= int64(MaxArgC) {
			valReg := fs.reserveReg()
			c.compileExprToReg(value, valReg)
			fs.emit(ABC(OP_SETI, tableReg, int(n.Value), valReg, 0), line)
		} else {
			keyReg := fs.reserveReg()
			c.compileExprToReg(t.Key, keyReg)
			valReg := fs.reserveReg()
			c.compileExprToReg(value, valReg)
			fs.emit(ABC(OP_SETTABLE, tableReg, keyReg, valReg, 0), line)
		}
		fs.freeReg = tableReg

	default:
		c.error(target, "invalid assignment target")
	}
}

// assignToTarget stores the value in srcReg into an assignment target
// (local, upvalue, global, field, or index).
func (c *compiler) assignToTarget(target ast.Expr, srcReg int, line int) {
	fs := c.fs

	switch t := target.(type) {
	case *ast.NameExpr:
		if reg, ok := fs.lookupLocal(t.Name); ok {
			if fs.isConst(t.Name) {
				c.error(target, "attempt to assign to const variable '%s'", t.Name)
				return
			}
			if reg != srcReg {
				fs.emit(ABC(OP_MOVE, reg, srcReg, 0, 0), line)
			}
			return
		}
		if idx, ok := c.resolveUpvalue(fs, t.Name); ok {
			if c.isConstUpvalue(fs, t.Name) {
				c.error(target, "attempt to assign to const variable '%s'", t.Name)
				return
			}
			fs.emit(ABC(OP_SETUPVAL, srcReg, idx, 0, 0), line)
			return
		}
		// Local _ENV: _ENV[name] via SETFIELD on local
		if envReg, ok := fs.lookupLocal("_ENV"); ok {
			nameK := fs.stringConstant(t.Name)
			fs.emitSetField(envReg, nameK, srcReg, line)
			return
		}
		// Global: _ENV[name]
		envUV := c.resolveEnv()
		nameK := fs.stringConstant(t.Name)
		fs.emitSetTabUp(envUV, nameK, srcReg, line)

	case *ast.FieldExpr:
		tableReg := fs.reserveReg()
		c.compileExprToReg(t.Table, tableReg)
		fieldK := fs.stringConstant(t.Field)
		fs.emitSetField(tableReg, fieldK, srcReg, line)
		fs.freeReg = tableReg

	case *ast.IndexExpr:
		tableReg := fs.reserveReg()
		c.compileExprToReg(t.Table, tableReg)
		if n, ok := t.Key.(*ast.NumberExpr); ok && n.Value >= 0 && n.Value <= int64(MaxArgC) {
			fs.emit(ABC(OP_SETI, tableReg, int(n.Value), srcReg, 0), line)
		} else {
			keyReg := fs.reserveReg()
			c.compileExprToReg(t.Key, keyReg)
			fs.emit(ABC(OP_SETTABLE, tableReg, keyReg, srcReg, 0), line)
		}
		fs.freeReg = tableReg

	default:
		c.error(target, "invalid assignment target")
	}
}

// compileSetGlobal compiles _ENV[name] = value.
func (c *compiler) compileSetGlobal(name string, value ast.Expr, line int) {
	fs := c.fs
	nameK := fs.stringConstant(name)
	tempReg := fs.reserveReg()
	c.compileExprToReg(value, tempReg)
	if envReg, ok := fs.lookupLocal("_ENV"); ok {
		fs.emitSetField(envReg, nameK, tempReg, line)
	} else {
		envUV := c.resolveEnv()
		fs.emitSetTabUp(envUV, nameK, tempReg, line)
	}
	fs.freeReg = tempReg
}

// ---------------------------------------------------------------------------
// Expression statements (function calls)
// ---------------------------------------------------------------------------

// compileExprStmt compiles a bare expression statement (must be a function call).
func (c *compiler) compileExprStmt(s *ast.ExprStmt) {
	fs := c.fs
	line := s.P.Line

	switch e := s.Expr.(type) {
	case *ast.FuncCallExpr:
		base := fs.freeReg
		c.compileFuncCall(e, base, 1, line) // 1 = discard results (C=1)
	case *ast.MethodCallExpr:
		base := fs.freeReg
		c.compileMethodCall(e, base, 1, line) // 1 = discard results
	default:
		c.error(s, "expression statement must be function call")
	}
}

// ---------------------------------------------------------------------------
// Return
// ---------------------------------------------------------------------------

// compileReturnStmt compiles "return expr, ..." — emits RETURN0, RETURN1,
// TAILCALL, or RETURN depending on the number and type of return values.
func (c *compiler) compileReturnStmt(s *ast.ReturnStmt) {
	fs := c.fs
	line := s.P.Line

	if len(s.Values) == 0 {
		fs.emit(ABC(OP_RETURN0, 0, 0, 0, 0), line)
		return
	}

	if len(s.Values) == 1 {
		// Check for tail call — cannot tail-call when there are to-be-closed
		// variables in scope, because they must be closed after the called
		// function returns. Captured upvalues are fine: OP_TAILCALL closes
		// them before entering the new frame.
		if !fs.needsCloseTBC(0) {
			if call, ok := s.Values[0].(*ast.FuncCallExpr); ok {
				base := fs.freeReg
				c.compileFuncCall(call, base, 0, line) // compile the call
				// Replace the CALL with TAILCALL
				lastPC := fs.pc() - 1
				inst := fs.proto.Code[lastPC]
				if inst.OpCode() == OP_CALL {
					fs.proto.Code[lastPC] = ABC(OP_TAILCALL, inst.A(), inst.B(), 0, 0)
				}
				return
			} else if call, ok := s.Values[0].(*ast.MethodCallExpr); ok {
				base := fs.freeReg
				c.compileMethodCall(call, base, 0, line) // compile the method call
				// Replace the CALL with TAILCALL
				lastPC := fs.pc() - 1
				inst := fs.proto.Code[lastPC]
				if inst.OpCode() == OP_CALL {
					fs.proto.Code[lastPC] = ABC(OP_TAILCALL, inst.A(), inst.B(), 0, 0)
				}
				return
			}
		}

		// Check for multi-return expression (vararg, method call)
		if isMultiRet(s.Values[0]) {
			base := fs.freeReg
			c.compileExprMultiRet(s.Values[0], 0)        // 0 = all results
			fs.emit(ABC(OP_RETURN, base, 0, 0, 0), line) // B=0 means return up to top
			return
		}

		reg := fs.freeReg
		c.compileExprToReg(s.Values[0], reg)
		fs.emit(ABC(OP_RETURN1, reg, 0, 0, 0), line)
		return
	}

	// Multiple return values
	base := fs.freeReg
	for i, val := range s.Values {
		if i == len(s.Values)-1 {
			// Last expr might be multi-return
			if isMultiRet(val) {
				c.compileExprMultiRet(val, 0)                // 0 = all results
				fs.emit(ABC(OP_RETURN, base, 0, 0, 0), line) // B=0 means return up to top
				return
			}
		}
		c.compileExprToReg(val, base+i)
		// Reset freeReg to reclaim temporaries (same fix as compileFuncCall)
		fs.freeReg = base + i + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}
	fs.emit(ABC(OP_RETURN, base, len(s.Values)+1, 0, 0), line)
}

// ---------------------------------------------------------------------------
// If/elseif/else
// ---------------------------------------------------------------------------

// compileIfStmt compiles "if cond then ... elseif ... else ... end".
func (c *compiler) compileIfStmt(s *ast.IfStmt) {
	fs := c.fs
	line := s.P.Line

	var exitJumps []int

	// Optimization: constant true condition — skip TEST+JMP, just compile body.
	// Matches Lua 5.4's luaK_goiftrue which emits no code for constant true.
	if isConstTrue(s.Cond) {
		fs.enterScope(false)
		c.compileBlock(s.Then)
		c.leaveScope(line)

		// Still need to jump past elseif/else blocks
		if len(s.ElseIfs) > 0 || s.Else != nil {
			exitJumps = append(exitJumps, fs.emitJump(line))
		}

		// Compile remaining branches (they're dead code but must be valid)
		for _, elif := range s.ElseIfs {
			eline := elif.P.Line
			c.compileCondJump(elif.Cond, false, eline)
			elifJump := fs.emitJump(eline)
			fs.enterScope(false)
			c.compileBlock(elif.Then)
			c.leaveScope(eline)
			exitJumps = append(exitJumps, fs.emitJump(eline))
			c.patchJump(elifJump)
		}
		if s.Else != nil {
			fs.enterScope(false)
			c.compileBlock(s.Else)
			c.leaveScope(line)
		}

		for _, jpc := range exitJumps {
			c.patchJump(jpc)
		}
		return
	}

	// Optimization: constant false/nil condition — emit JMP over then-block.
	// Matches Lua 5.4's luaK_goiffalse which emits only a JMP for constant false.
	if isConstFalsy(s.Cond) {
		thenJump := fs.emitJump(line) // unconditional jump past then-block
		fs.enterScope(false)
		c.compileBlock(s.Then)
		c.leaveScope(line)

		if len(s.ElseIfs) > 0 || s.Else != nil {
			exitJumps = append(exitJumps, fs.emitJump(line))
		}
		c.patchJump(thenJump)

		for _, elif := range s.ElseIfs {
			eline := elif.P.Line
			c.compileCondJump(elif.Cond, false, eline)
			elifJump := fs.emitJump(eline)
			fs.enterScope(false)
			c.compileBlock(elif.Then)
			c.leaveScope(eline)
			exitJumps = append(exitJumps, fs.emitJump(eline))
			c.patchJump(elifJump)
		}
		if s.Else != nil {
			fs.enterScope(false)
			c.compileBlock(s.Else)
			c.leaveScope(line)
		}

		for _, jpc := range exitJumps {
			c.patchJump(jpc)
		}
		return
	}

	// General case: emit TEST + JMP
	c.compileCondJump(s.Cond, false, line)
	thenJump := fs.emitJump(line) // jump past then-block if false
	fs.enterScope(false)
	c.compileBlock(s.Then)
	c.leaveScope(line)

	// Jump past all else/elseif blocks after then
	if len(s.ElseIfs) > 0 || s.Else != nil {
		exitJumps = append(exitJumps, fs.emitJump(line))
	}

	c.patchJump(thenJump)

	// elseif branches
	for _, elif := range s.ElseIfs {
		eline := elif.P.Line
		c.compileCondJump(elif.Cond, false, eline)
		elifJump := fs.emitJump(eline)
		fs.enterScope(false)
		c.compileBlock(elif.Then)
		c.leaveScope(eline)
		exitJumps = append(exitJumps, fs.emitJump(eline))
		c.patchJump(elifJump)
	}

	// else
	if s.Else != nil {
		fs.enterScope(false)
		c.compileBlock(s.Else)
		c.leaveScope(line)
	}

	// Patch all exit jumps to here
	for _, jpc := range exitJumps {
		c.patchJump(jpc)
	}
}

// compileCondJump compiles an expression and emits a conditional jump.
// If jumpOnFalse is true, it jumps when the condition is false.
func (c *compiler) compileCondJump(cond ast.Expr, jumpOnFalse bool, line int) {
	fs := c.fs
	reg := fs.freeReg
	c.compileExprToReg(cond, reg)
	k := 1 // skip if truthy (jump on false)
	if !jumpOnFalse {
		k = 0 // skip if falsy (jump on true means we fall through on true)
	}
	fs.emit(ABC(OP_TEST, reg, 0, 0, k), line)
	// Free the condition register — it's a temporary that is only needed for
	// the TEST instruction above. Without this, the register "leaks" and
	// subsequent locals in the then/body block get allocated to higher
	// registers, causing debug.getlocal to return wrong values (the local
	// name maps to the condition's register instead of the local's actual one).
	fs.freeReg = reg
}

// ---------------------------------------------------------------------------
// While
// ---------------------------------------------------------------------------

// isConstTrue reports whether expr is the literal constant `true`.
// Used by compileWhileStmt to optimize `while true do ... end` into
// an unconditional loop (no condition test), matching reference Lua 5.4.
func isConstTrue(expr ast.Expr) bool {
	_, ok := expr.(*ast.TrueExpr)
	return ok
}

// isConstFalsy reports whether expr is the literal `false` or `nil`.
// Used for constant-folding conditions in if/while statements.
func isConstFalsy(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.FalseExpr, *ast.NilExpr:
		return true
	}
	return false
}

// compileWhileStmt compiles "while cond do ... end".
func (c *compiler) compileWhileStmt(s *ast.WhileStmt) {
	fs := c.fs
	line := s.P.Line

	// Optimization: `while true do ... end` emits no condition test,
	// matching reference Lua 5.4. The loop body executes unconditionally
	// and the back-jump targets the first body instruction directly.
	if isConstTrue(s.Cond) {
		loopStart := fs.pc()
		fs.enterScope(true)

		// Body
		c.compileBlock(s.Body)

		// Close upvalues for body locals before jumping back.
		scope := fs.scopes[len(fs.scopes)-1]
		if fs.nActVar > scope.nLocals {
			fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), line)
		}

		// Jump back to loop start (unconditional)
		backJump := fs.emitJump(line)
		offset := loopStart - (fs.pc()) // negative
		if offset > MaxSJ || offset < MinSJ {
			c.error(nil, "control structure too long")
		} else {
			fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)
		}

		c.leaveScope(line)
		return
	}

	loopStart := fs.pc()
	fs.enterScope(true)

	// Test condition
	c.compileCondJump(s.Cond, false, line)
	exitJump := fs.emitJump(line)

	// Body
	c.compileBlock(s.Body)

	// Close upvalues for body locals before jumping back.
	// This ensures each iteration gets its own closed upvalue copy.
	scope := fs.scopes[len(fs.scopes)-1]
	if fs.nActVar > scope.nLocals {
		fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), line)
	}

	// Jump back to condition
	backJump := fs.emitJump(line)
	offset := loopStart - (fs.pc()) // negative
	if offset > MaxSJ || offset < MinSJ {
		c.error(nil, "control structure too long")
	} else {
		fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)
	}

	c.leaveScope(line)
	c.patchJump(exitJump)
}

// ---------------------------------------------------------------------------
// Repeat
// ---------------------------------------------------------------------------

// compileRepeatStmt compiles "repeat ... until cond".
func (c *compiler) compileRepeatStmt(s *ast.RepeatStmt) {
	fs := c.fs
	line := s.P.Line

	loopStart := fs.pc()
	fs.enterScope(true)

	// Use labelEndOpt=false because the until condition can reference
	// body locals -- labels at the end of a repeat block must NOT treat
	// preceding locals as out of scope.
	c.compileBlockWith(s.Body, false, s.Cond.Pos().Line)

	// Evaluate condition (may reference body locals)
	condLine := s.Cond.Pos().Line
	reg := fs.freeReg
	c.compileExprToReg(s.Cond, reg)

	// Close upvalues for body locals. OP_CLOSE captures values but does
	// not clear stack slots, so the condition result in reg remains valid.
	scope := fs.scopes[len(fs.scopes)-1]
	if fs.nActVar > scope.nLocals {
		fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), condLine)
	}

	// If condition is falsy, jump back (keep looping)
	fs.emit(ABC(OP_TEST, reg, 0, 0, 0), condLine) // skip next if truthy → exit
	backJump := fs.emitJump(condLine)             // jump back (cond is false)
	offset := loopStart - fs.pc()
	if offset > MaxSJ || offset < MinSJ {
		c.error(nil, "control structure too long")
	} else {
		fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)
	}

	// Fall through here when condition is true (exit loop)
	c.leaveScope(line)
}

// ---------------------------------------------------------------------------
// Do block
// ---------------------------------------------------------------------------

// compileDoStmt compiles "do ... end" — creates a new scope for the body.
func (c *compiler) compileDoStmt(s *ast.DoStmt) {
	c.fs.enterScope(false)
	c.compileBlock(s.Body)
	endLine := s.EndLine
	if endLine == 0 {
		endLine = s.P.Line
	}
	c.leaveScope(endLine)
}

// ---------------------------------------------------------------------------
// Numeric for
// ---------------------------------------------------------------------------

// compileForNumStmt compiles "for i = start, stop, step do ... end".
// Uses FORPREP/FORLOOP opcodes with 4 internal registers (init, limit, step, i).
func (c *compiler) compileForNumStmt(s *ast.ForNumStmt) {
	fs := c.fs
	line := s.P.Line

	fs.enterScope(true)

	// Reserve 4 registers: (internal) init, limit, step, (external) i
	base := fs.freeReg

	// Compile init, limit, step into base, base+1, base+2
	c.compileExprToReg(s.Start, base)
	fs.freeReg = base + 1
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}

	c.compileExprToReg(s.Stop, base+1)
	fs.freeReg = base + 2
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}

	if s.Step != nil {
		c.compileExprToReg(s.Step, base+2)
	} else {
		fs.emit(AsBx(OP_LOADI, base+2, 1), line) // default step = 1
	}
	fs.freeReg = base + 3
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}

	// FORPREP — jumps to FORLOOP if not to run
	forPrepPC := fs.emit(ABx(OP_FORPREP, base, 0), line)

	// The loop variable is at base+3
	fs.freeReg = base + 4
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	fs.checkRegLimit()

	// Add internal for loop variables as hidden locals to protect their registers
	// This ensures freeReg won't be reset below base+4 during the loop body
	fs.checkVarLimitAt(4, line, "for")
	fs.locals = append(fs.locals,
		localVar{name: "(for state)", reg: base, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 1, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 2, startPC: fs.pc()},
	)
	fs.nActVar += 3

	// Add the loop variable as a local
	fs.locals = append(fs.locals, localVar{
		name:    s.Name.Name,
		reg:     base + 3,
		startPC: fs.pc(),
	})
	fs.nActVar++

	// Body
	c.compileBlock(s.Body)

	// Close upvalues at the loop variable (base+3) and above before looping back,
	// but only if any variable from the loop variable onwards was captured by an
	// inner closure or has a <close> attribute. Skipping CLOSE when not needed
	// avoids inflating instruction counts (which affects debug count hooks).
	needClose := false
	for i := len(fs.locals) - 1; i >= 0; i-- {
		if fs.locals[i].reg < base+3 {
			break
		}
		if fs.locals[i].captured || fs.locals[i].attrib == "close" {
			needClose = true
			break
		}
	}
	if needClose {
		fs.emit(ABC(OP_CLOSE, base+3, 0, 0, 0), line)
	}

	// FORLOOP — jumps back to just after FORPREP
	loopPC := fs.emit(ABx(OP_FORLOOP, base, 0), line)

	// Patch FORPREP to jump to FORLOOP
	bodyLen := loopPC - forPrepPC - 1
	if bodyLen > MaxArgBx {
		c.error(nil, "control structure too long")
	}
	fs.proto.Code[forPrepPC] = fs.proto.Code[forPrepPC].SetBx(bodyLen)

	// Patch FORLOOP to jump back
	fs.proto.Code[loopPC] = fs.proto.Code[loopPC].SetBx(bodyLen)

	c.leaveScope(line)
}

// ---------------------------------------------------------------------------
// Generic for
// ---------------------------------------------------------------------------

// compileForInStmt compiles "for k, v in iter do ... end".
// Uses TFORPREP/TFORCALL/TFORLOOP with 4 control registers + loop variables.
func (c *compiler) compileForInStmt(s *ast.ForInStmt) {
	fs := c.fs
	line := s.P.Line

	fs.enterScope(true)

	// Reserve 4 control registers: iterator, state, control, closing
	base := fs.freeReg

	// Compile iterator expressions into base, base+1, base+2, base+3
	// The 4th value (base+3) is the to-be-closed variable per Lua 5.4
	nIter := len(s.Iters)
	if nIter == 1 && isMultiRet(s.Iters[0]) {
		// Single multi-return expression (e.g., pairs(t)) - ask for 4 results
		c.compileExprMultiRet(s.Iters[0], 4)
		fs.freeReg = base + 4
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	} else {
		for i, iter := range s.Iters {
			if i < 4 {
				c.compileExprToReg(iter, base+i)
				fs.freeReg = base + i + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			}
		}
		// Fill missing with nil
		for i := nIter; i < 4; i++ {
			fs.emit(ABC(OP_LOADNIL, base+i, 0, 0, 0), line)
			fs.freeReg = base + i + 1
			if fs.freeReg > fs.maxReg {
				fs.maxReg = fs.freeReg
			}
		}
	}
	fs.freeReg = base + 4
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}

	// Add internal for-in variables as hidden locals to protect their registers.
	// These must be registered BEFORE OP_TBC so localName() can resolve the
	// variable name at the TBC instruction's PC.
	localStartPC := fs.pc()
	fs.checkVarLimitAt(4+len(s.Names), line, "in")
	fs.locals = append(fs.locals,
		localVar{name: "(for state)", reg: base, startPC: localStartPC},
		localVar{name: "(for state)", reg: base + 1, startPC: localStartPC},
		localVar{name: "(for state)", reg: base + 2, startPC: localStartPC},
		localVar{name: "(for state)", reg: base + 3, startPC: localStartPC, attrib: "close"},
	)
	fs.nActVar += 4

	// Mark base+3 as to-be-closed (the 4th return from iterator factory)
	fs.emit(ABC(OP_TBC, base+3, 0, 0, 0), line)

	// TFORPREP
	tforPrepPC := fs.emit(ABx(OP_TFORPREP, base, 0), line)

	// Loop variables start at base+4
	nVars := len(s.Names)
	for i, name := range s.Names {
		reg := base + 4 + i
		fs.freeReg = reg + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
		fs.locals = append(fs.locals, localVar{
			name:    name.Name,
			reg:     reg,
			startPC: fs.pc(),
		})
		fs.nActVar++
	}
	fs.checkRegLimit()

	// Body
	c.compileBlock(s.Body)

	// Close upvalues at the loop variables (base+4) and above before next iteration.
	// This ensures each iteration gets its own closed upvalue copy.
	fs.emit(ABC(OP_CLOSE, base+4, 0, 0, 0), line)

	// TFORCALL — calls the iterator
	tforCallPC := fs.emit(ABC(OP_TFORCALL, base, 0, nVars, 0), line)
	_ = tforCallPC

	// TFORLOOP — checks if control variable is nil
	tforLoopPC := fs.emit(ABx(OP_TFORLOOP, base, 0), line)

	// Patch TFORPREP to jump to TFORCALL
	bodyLen := tforCallPC - tforPrepPC - 1
	if bodyLen > MaxArgBx {
		c.error(nil, "control structure too long")
	}
	fs.proto.Code[tforPrepPC] = fs.proto.Code[tforPrepPC].SetBx(bodyLen)

	// Patch TFORLOOP to jump back to loop body (after TFORPREP)
	backLen := tforLoopPC - tforPrepPC - 1
	if backLen > MaxArgBx {
		c.error(nil, "control structure too long")
	}
	fs.proto.Code[tforLoopPC] = fs.proto.Code[tforLoopPC].SetBx(backLen)

	// Fix loop variable visibility: they should NOT be visible during the
	// TFORCALL iterator invocation (matching Lua 5.4 where loop vars start
	// after TFORCALL). Emit separate debug entries for the body range only.
	bodyEnd := tforCallPC // exclusive: body ends just before TFORCALL
	loopVarStart := len(fs.locals) - nVars
	for i := 0; i < nVars; i++ {
		lv := &fs.locals[loopVarStart+i]
		fs.proto.Locals = append(fs.proto.Locals, LocalVar{
			Name:    lv.name,
			StartPC: lv.startPC,
			EndPC:   bodyEnd,
		})
		// Mark as already emitted so leaveScope doesn't emit again
		lv.startPC = -1
	}

	c.leaveScope(line)
}

// ---------------------------------------------------------------------------
// Break / Goto / Labels
// ---------------------------------------------------------------------------

// compileBreakStmt compiles "break" — emits OP_CLOSE if needed, then a
// forward jump to be patched when the enclosing loop scope exits.
func (c *compiler) compileBreakStmt(s *ast.BreakStmt) {
	fs := c.fs
	scope := fs.findLoopScope()
	if scope == nil {
		c.errorAtEOF("break outside loop at line %d", s.Pos().Line)
		return
	}
	// Emit OP_CLOSE if there are close/captured locals being exited
	if fs.needsClose(scope.nLocals) {
		fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), s.P.Line)
	}
	jpc := fs.emitJump(s.P.Line)
	scope.breakJumps = append(scope.breakJumps, jpc)
}

// compileGotoStmt compiles "goto label". For backward gotos the jump is
// resolved immediately; forward gotos are recorded and patched at the label.
func (c *compiler) compileGotoStmt(s *ast.GotoStmt) {
	fs := c.fs
	line := s.P.Line

	// Check if label already exists (backward goto)
	for _, lbl := range fs.labels {
		if lbl.name == s.Label {
			// Emit OP_CLOSE if exiting scope with TBC/captured locals
			if fs.needsClose(lbl.nLocals) {
				fs.emit(ABC(OP_CLOSE, fs.regBaseForLocals(lbl.nLocals), 0, 0, 0), line)
			}
			jpc := fs.emitJump(line)
			offset := lbl.pc - (jpc + 1)
			if offset > MaxSJ || offset < MinSJ {
				c.error(nil, "control structure too long")
			} else {
				fs.proto.Code[jpc] = fs.proto.Code[jpc].SetSJ(offset)
			}
			return
		}
	}

	// Forward goto — emit placeholder OP_CLOSE and record it.
	// The OP_CLOSE operand will be patched when the label is resolved.
	closePC := -1
	if fs.needsClose(0) {
		// There are TBC/captured locals somewhere; emit placeholder
		closePC = fs.emit(ABC(OP_CLOSE, fs.regTop(), 0, 0, 0), line)
	}
	jpc := fs.emitJump(line)
	fs.pendGotos = append(fs.pendGotos, pendingGoto{
		name:    s.Label,
		pc:      jpc,
		nLocals: fs.nActVar,
		line:    line,
		closePC: closePC,
	})
}

// compileLabelStmt compiles "::label::" — records the label and resolves
// any pending forward gotos that target it.
//
// When atBlockEnd is true, the label is the last non-label statement in
// its block (Lua 5.4 §3.3.4). In that case, locals declared before the
// label are treated as already out of scope, allowing goto to jump past
// them.
func (c *compiler) compileLabelStmt(s *ast.LabelStmt, atBlockEnd bool, afterLine int) {
	fs := c.fs

	// Check for duplicate label in all visible scopes. Labels are
	// removed from fs.labels when their scope exits, so all entries
	// are from currently active (enclosing) scopes.
	scope := fs.scopes[len(fs.scopes)-1]
	for _, lbl := range fs.labels {
		if lbl.name == s.Name {
			// Use the duplicate label's EndLine for the error prefix,
			// matching Lua 5.4's checkrepeated() which uses ls->linenumber
			// (the line after the closing :: of the duplicate label).
			c.error(s.EndLine, "label '%s' already defined on line %d", s.Name, lbl.line)
			return
		}
	}

	// Determine effective nLocals for this label. If the label is at
	// the end of a block, locals in the current scope are about to go
	// out of scope, so use the enclosing scope's nLocals instead.
	labelNLocals := fs.nActVar
	if atBlockEnd {
		labelNLocals = scope.nLocals
	}

	fs.labels = append(fs.labels, labelInfo{
		name:    s.Name,
		pc:      fs.pc(),
		line:    s.P.Line,
		nLocals: labelNLocals,
	})

	// Resolve pending gotos from the current scope level only. Gotos from
	// enclosing scopes cannot jump into this block (Lua 5.4 §3.3.4).
	remaining := fs.pendGotos[:0]
	for i, pg := range fs.pendGotos {
		if i >= scope.firstGoto && pg.name == s.Name {
			// Validate: goto must not jump into scope of a local variable
			if pg.nLocals < labelNLocals {
				// Find the name of the first variable the goto jumps over
				varName := "?"
				baseIdx := len(fs.locals) - (fs.nActVar - pg.nLocals)
				if baseIdx >= 0 && baseIdx < len(fs.locals) {
					varName = fs.locals[baseIdx].name
				}
				c.errorAtLine(afterLine, "<goto %s> at line %d jumps into the scope of local '%s'", pg.name, pg.line, varName)
				remaining = append(remaining, pg)
				continue
			}
			// Patch placeholder OP_CLOSE if one was emitted
			if pg.closePC >= 0 && labelNLocals < pg.nLocals {
				fs.proto.Code[pg.closePC] = fs.proto.Code[pg.closePC].SetA(fs.regBaseForLocals(labelNLocals))
			}
			offset := fs.pc() - (pg.pc + 1)
			if offset > MaxSJ || offset < MinSJ {
				c.error(nil, "control structure too long")
			} else {
				fs.proto.Code[pg.pc] = fs.proto.Code[pg.pc].SetSJ(offset)
			}
		} else {
			remaining = append(remaining, pg)
		}
	}
	fs.pendGotos = remaining
}

// ---------------------------------------------------------------------------
// Function statements
// ---------------------------------------------------------------------------

// compileFuncStmt compiles "function name(...) ... end" or "function a.b.c(...) ... end".
func (c *compiler) compileFuncStmt(s *ast.FuncStmt) {
	fs := c.fs
	line := s.P.Line

	// Compile the function body
	protoIdx := c.compileFunc(s.Func, line)

	// Create closure in a temp register
	reg := fs.reserveReg()
	fs.emit(ABx(OP_CLOSURE, reg, protoIdx), line)

	// Assign to the target
	switch name := s.Name.(type) {
	case *ast.NameExpr:
		// Simple name: could be local, upvalue, or global
		if localReg, ok := fs.lookupLocal(name.Name); ok {
			if fs.isConst(name.Name) {
				c.error(s.Name, "attempt to assign to const variable '%s'", name.Name)
				fs.freeReg = reg
				return
			}
			fs.emit(ABC(OP_MOVE, localReg, reg, 0, 0), line)
		} else if uvIdx, ok := c.resolveUpvalue(fs, name.Name); ok {
			if c.isConstUpvalue(fs, name.Name) {
				c.error(s.Name, "attempt to assign to const variable '%s'", name.Name)
				fs.freeReg = reg
				return
			}
			fs.emit(ABC(OP_SETUPVAL, reg, uvIdx, 0, 0), line)
		} else if envReg, ok := fs.lookupLocal("_ENV"); ok {
			nameK := fs.stringConstant(name.Name)
			fs.emitSetField(envReg, nameK, reg, line)
		} else {
			envUV := c.resolveEnv()
			nameK := fs.stringConstant(name.Name)
			fs.emitSetTabUp(envUV, nameK, reg, line)
		}

	case *ast.FieldExpr:
		// Dotted name: a.b.c = function ...
		tableReg := fs.reserveReg()
		c.compileExprToReg(name.Table, tableReg)
		fieldK := fs.stringConstant(name.Field)
		fs.emitSetField(tableReg, fieldK, reg, line)
		fs.freeReg = tableReg
	}

	fs.freeReg = reg
}

// compileLocalFuncStmt compiles "local function name(...) ... end".
// The local is registered before the body so the function can reference itself.
func (c *compiler) compileLocalFuncStmt(s *ast.LocalFuncStmt) {
	fs := c.fs
	line := s.P.Line

	// Register the local first (allows recursion)
	// Use "(" as near token: Lua 5.4 registers the local for f after parsing
	// the opening parenthesis, so the error reports near '(' not near 'f'.
	fs.checkVarLimitAt(1, line, "(")
	localIdx := len(fs.locals)
	reg := fs.freeReg
	fs.locals = append(fs.locals, localVar{
		name:    s.Name.Name,
		reg:     reg,
		startPC: fs.pc(),
	})
	fs.nActVar++
	fs.freeReg++
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	fs.checkRegLimit()
	_ = localIdx

	protoIdx := c.compileFunc(s.Func, line)
	fs.emit(ABx(OP_CLOSURE, reg, protoIdx), line)
}

// compileGlobalStmt compiles "global name = expr" — assigns to _ENV[name].
func (c *compiler) compileGlobalStmt(s *ast.GlobalStmt) {
	// Treat like assignment to _ENV[name]
	fs := c.fs
	line := s.P.Line

	if s.Star {
		return // global * is a parser directive, no codegen
	}

	for i, name := range s.Names {
		nameK := fs.stringConstant(name.Name)
		if i < len(s.Values) {
			reg := fs.reserveReg()
			c.compileExprToReg(s.Values[i], reg)
			if envReg, ok := fs.lookupLocal("_ENV"); ok {
				fs.emitSetField(envReg, nameK, reg, line)
			} else {
				envUV := c.resolveEnv()
				fs.emitSetTabUp(envUV, nameK, reg, line)
			}
			fs.freeReg = reg
		}
	}
}

// compileGlobalFuncStmt compiles "global function name(...) ... end".
func (c *compiler) compileGlobalFuncStmt(s *ast.GlobalFuncStmt) {
	fs := c.fs
	line := s.P.Line

	protoIdx := c.compileFunc(s.Func, line)
	reg := fs.reserveReg()
	fs.emit(ABx(OP_CLOSURE, reg, protoIdx), line)

	nameK := fs.stringConstant(s.Name.Name)
	if envReg, ok := fs.lookupLocal("_ENV"); ok {
		fs.emitSetField(envReg, nameK, reg, line)
	} else {
		envUV := c.resolveEnv()
		fs.emitSetTabUp(envUV, nameK, reg, line)
	}

	fs.freeReg = reg
}

// ---------------------------------------------------------------------------
// Function compilation (body)
// ---------------------------------------------------------------------------

// compileFunc compiles a function body (shared by function expressions,
// function statements, and local function statements). Returns the child
// proto index for use with OP_CLOSURE.
func (c *compiler) compileFunc(fe *ast.FuncExpr, line int) int {
	parentFS := c.fs

	source := parentFS.proto.Source
	fs := c.newFuncState(source, parentFS)
	fs.maxReg = 2

	fs.proto.LineDef = fe.P.Line
	fs.proto.NumParams = len(fe.Params)
	fs.proto.IsVarArg = fe.VarArg

	// _ENV is NOT unconditionally captured. It will be resolved lazily
	// by resolveEnv() only when the function body actually references a
	// global variable. This avoids inflating nups for functions that don't
	// use globals, matching reference Lua 5.4 behavior.

	fs.enterScope(false)

	// Parameters are local variables.
	// Register all params first, then check the limit with near ')' to
	// match Lua 5.4 which checks after consuming the closing parenthesis.
	for _, param := range fe.Params {
		reg := fs.freeReg
		fs.locals = append(fs.locals, localVar{
			name:    param.Name,
			reg:     reg,
			startPC: 0,
		})
		fs.nActVar++
		fs.freeReg++
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	}
	fs.checkVarLimitAt(0, fe.P.Line, ")")
	fs.checkRegLimit()

	// Vararg prep
	if fe.VarArg {
		fs.emit(ABC(OP_VARARGPREP, fs.proto.NumParams, 0, 0, 0), line)
	}

	c.compileBlockWith(fe.Body, true, fe.EndLine)

	// Ensure function ends with a return.
	// Use the 'end' keyword line if available (matches Lua 5.4 for __close errors).
	lastLine := line
	if fe.EndLine > 0 {
		lastLine = fe.EndLine
	} else if fe.Body != nil && len(fe.Body.Stmts) > 0 {
		lastLine = fe.Body.Stmts[len(fe.Body.Stmts)-1].Pos().Line
	}
	fs.proto.LastLine = lastLine

	// Check if last instruction is already a return
	needReturn := true
	if len(fs.proto.Code) > 0 {
		last := fs.proto.Code[len(fs.proto.Code)-1].OpCode()
		if last == OP_RETURN || last == OP_RETURN0 || last == OP_RETURN1 || last == OP_TAILCALL {
			needReturn = false
		}
	}
	if needReturn {
		fs.emit(ABC(OP_RETURN0, 0, 0, 0, 0), lastLine)
	}

	c.leaveScope(lastLine)

	proto := c.closeFuncState()
	return parentFS.addProto(proto)
}
