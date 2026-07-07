package compiler

import (
	"github.com/iceisfun/golua/ast"
)

// stmtEndLine returns the "end line" of a statement — the line of the closing
// 'end' keyword for block statements, or the start line for simple statements.
// This matches Lua 5.4's ls->lastline which tracks the last consumed token.
func stmtEndLine(s ast.Stmt) int {
	switch s := s.(type) {
	case *ast.IfStmt:
		return s.EndLine
	case *ast.WhileStmt:
		return s.EndLine
	case *ast.DoStmt:
		return s.EndLine
	case *ast.ForNumStmt:
		return s.EndLine
	case *ast.ForInStmt:
		return s.EndLine
	case *ast.RepeatStmt:
		// repeat..until has no 'end'; use the condition's line
		return s.Cond.Pos().Line
	case *ast.ExprStmt:
		return s.Expr.Pos().Line
	default:
		return s.Pos().Line
	}
}

// preRegisterUpvalues walks an assignment target expression and pre-registers
// any upvalue references in left-to-right order. This ensures that upvalue
// indices match Lua 5.4's source order, where LHS targets are processed
// before RHS values during compilation.
func (c *compiler) preRegisterUpvalues(fs *funcState, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.NameExpr:
		// Inlined `<const>` locals are substituted at use-sites and
		// never become upvalues.
		if _, isInlined := lookupInlinedAny(fs, e.Name); isInlined {
			return
		}
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
	fs.addUpvalue(envUpvalueName, true, 0)

	fs.enterScope(false)

	// Emit VARARGPREP
	line := 0
	if block != nil && len(block.Stmts) > 0 {
		line = block.Start.Line
	}
	fs.emit(ABC(OP_VARARGPREP, 0, 0, 0, 0), line)

	c.compileBlock(block)

	// Emit final return — use the end line of the last statement (e.g., the
	// 'end' keyword line for if/while/do/for/repeat), matching Lua 5.4's
	// close_func which uses ls->lastline (the last token consumed).
	lastLine := line
	if block != nil && len(block.Stmts) > 0 {
		lastLine = stmtEndLine(block.Stmts[len(block.Stmts)-1])
	}
	fs.emit(ABC(OP_RETURN0, 0, 0, 0, 0), lastLine)
	// LastLine stays 0 for the main chunk (set at proto init).
	// Lua 5.4 always reports lastlinedefined=0 for the top-level function.
	fs.closeLine = lastLine

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

			// Compute afterLine for the entire label run. Lua's labelstat()
			// consumes any trailing no-op statements (';' and following labels)
			// via its `while (token == ';' || token == TK_DBCOLON)` loop BEFORE
			// it resolves pending gotos. So `ls->lastline` — and therefore the
			// "jumps into scope" error line — is the line of the first real
			// (non-label, non-empty) statement, not the first semicolon. Scan
			// past both labels and empty statements to find that line.
			afterEnd := runEnd
			for afterEnd < len(stmts) {
				switch stmts[afterEnd].(type) {
				case *ast.LabelStmt, *ast.EmptyStmt:
					afterEnd++
					continue
				}
				break
			}
			afterLine := blockAfterLine
			if afterLine == 0 {
				afterLine = c.endLine
			}
			if afterEnd < len(stmts) {
				afterLine = stmts[afterEnd].Pos().Line
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
		// Record this statement's AST end line so a block-exit OP_CLOSE can be
		// attributed to the block's last statement (reference Lua ls->lastline).
		// Use the AST End() span directly (not stmtEndLine, whose older form
		// here returns a statement's start line) so multi-line statements report
		// their closing token's line.
		if lbl, ok := stmt.(*ast.LabelStmt); ok && lbl.EndLine != 0 {
			c.fs.blockLastLine = lbl.EndLine
		} else {
			c.fs.blockLastLine = stmt.End().Line
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

	// Lua 5.5 inlining of `<const>` locals (lparser.c:localstat):
	// when nNames == nValues and the LAST variable is `<const>` and its
	// initializer folds to a compile-time scalar constant (nil/bool/
	// number/string), we skip emitting code for it, do not allocate a
	// register for it, and substitute its value at use-sites.
	//
	// Only the last variable is eligible (matching the reference
	// compiler), and the inlining only applies when there are no
	// adjustments (no missing/extra values).
	inlineLast := false
	var inlineVal Value
	if nNames == nValues && nNames > 0 {
		lastIdx := nNames - 1
		lastAttrib := ""
		if lastIdx < len(s.Attribs) {
			lastAttrib = s.Attribs[lastIdx]
		}
		// _ENV is the implicit table for global resolution. Reference
		// Lua does inline `local _ENV <const> = K` (and the resulting
		// `name = v` indexes the inlined constant), but plumbing the
		// inlined value through every global access site here would
		// touch many call sites for negligible benefit. We keep _ENV
		// allocated to a register so the existing `lookupLocal(envUpvalueName)`
		// fallback paths continue to work; runtime semantics still
		// match (assigning a global indexes the local _ENV value).
		if lastAttrib == attribConst && s.Names[lastIdx].Name != envUpvalueName {
			if v, ok := c.tryFoldConstScalar(s.Values[lastIdx]); ok {
				inlineLast = true
				inlineVal = v
			}
		}
	}

	// Number of variables that actually consume a register (everything
	// except the optionally-inlined trailing var).
	nRegVars := nNames
	if inlineLast {
		nRegVars--
	}

	// Compile all RHS values into base, base+1, ... (skipping the
	// trailing inlined value, which produces no code).
	lastIsMultiRet := false
	if nValues > 0 {
		for i := 0; i < nValues; i++ {
			if inlineLast && i == nValues-1 {
				// Inlined trailing value — no code, no register.
				continue
			}
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
				fs.freeReg = base + nRegVars // discard temp
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

	fs.freeReg = base + nRegVars
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
		isInlined := inlineLast && i == nNames-1
		reg := base + i
		if isInlined {
			reg = -1
		}
		fs.locals = append(fs.locals, localVar{
			name:      name.Name,
			reg:       reg,
			startPC:   -1,
			attrib:    attrib,
			inlined:   isInlined,
			inlineVal: inlineValIf(isInlined, inlineVal),
		})
		fs.nActVar++
	}

	// Activate all locals at the current PC
	for i := 0; i < nNames; i++ {
		fs.locals[baseIdx+i].startPC = fs.pc()
	}

	// Emit OP_TBC for <close> variables
	for i := 0; i < nNames; i++ {
		if fs.locals[baseIdx+i].attrib == attribClose {
			fs.emit(ABC(OP_TBC, base+i, 0, 0, 0), line)
		}
	}
}

// inlineValIf returns v when cond is true, otherwise the zero Value.
// Used purely as a struct-literal helper to keep the inlined-local
// declaration site compact.
func inlineValIf(cond bool, v Value) Value {
	if cond {
		return v
	}
	return Value{}
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
	// LHS table/key sub-expressions of indexed targets are resolved before the
	// RHS. A bare-local table/key operand uses its live register directly
	// (matching reference Lua), EXCEPT when a later target reassigns that local
	// — then it is snapshotted into a temp to preserve its original value
	// (reference's check_conflict). Example: i, a[i], a = j, i, i  — a[i]'s
	// table 'a' is reassigned by the third target, so it must use a snapshot of
	// the original 'a'.

	// Phase 0: Pre-register upvalues referenced by LHS targets in
	// left-to-right order. This ensures upvalue indices match Lua 5.4's
	// source order, where targets are processed before RHS values.
	for i := 0; i < nTargets; i++ {
		c.preRegisterUpvalues(fs, s.Targets[i])
	}

	// assignedReg[i] is the local register a bare-local target reassigns, or -1.
	// Used to detect check_conflict-style conflicts with earlier indexed
	// targets' live local table/key operands.
	assignedReg := make([]int, nTargets)
	for i := 0; i < nTargets; i++ {
		assignedReg[i] = -1
		if ne, ok := s.Targets[i].(*ast.NameExpr); ok {
			if _, inlined := lookupInlinedAny(fs, ne.Name); !inlined {
				if reg, ok := fs.lookupLocal(ne.Name); ok {
					assignedReg[i] = reg
				}
			}
		}
	}
	conflictsLaterThan := func(after, reg int) bool {
		for j := after + 1; j < nTargets; j++ {
			if assignedReg[j] == reg {
				return true
			}
		}
		return false
	}

	// Phase 1: Resolve LHS indexed targets' table/key operands.
	type precomputedTarget struct {
		tableReg int // reg holding table reference (live local or temp)
		keyReg   int // reg holding key (-1 for field/intKey targets)
		fieldK   int // constant index for field targets (-1 otherwise)
		intKey   int // constant integer key for SETI (-1 if not applicable)
	}
	precomputed := make([]precomputedTarget, nTargets)
	tempBase := fs.freeReg

	for i := 0; i < nTargets; i++ {
		i := i // capture for the conflict closure below
		conflict := func(reg int) bool { return conflictsLaterThan(i, reg) }
		switch t := s.Targets[i].(type) {
		case *ast.IndexExpr:
			tReg := c.indexAssignOperand(t.Table, conflict)
			if n, ok := t.Key.(*ast.NumberExpr); ok && n.Value >= 0 && n.Value <= int64(MaxArgC) {
				precomputed[i] = precomputedTarget{tableReg: tReg, keyReg: -1, fieldK: -1, intKey: int(n.Value)}
			} else {
				kReg := c.indexAssignOperand(t.Key, conflict)
				precomputed[i] = precomputedTarget{tableReg: tReg, keyReg: kReg, fieldK: -1, intKey: -1}
			}
		case *ast.FieldExpr:
			tReg := c.indexAssignOperand(t.Table, conflict)
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
				// Reserve the result registers (valBase+i .. valBase+nTargets-1).
				// compileExprMultiRet emits the CALL/VARARG but does not bump
				// freeReg, so without this the high-water mark (MaxStack) omits
				// these slots. A store that triggers a metamethod (e.g.
				// __newindex) would then place its call frame on top of the
				// still-pending value registers and clobber them.
				fs.freeReg = valBase + nTargets
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
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
			// Pre-evaluated indexed/field target. Emit the store at the target's
			// own field/key line (not the statement line) so a runtime index
			// error on it reports the right line, matching reference Lua.
			storeLine := exprEndLine(s.Targets[i])
			if pc.intKey >= 0 {
				fs.emit(ABC(OP_SETI, pc.tableReg, pc.intKey, valBase+i, 0), storeLine)
			} else if pc.keyReg >= 0 {
				fs.emit(ABC(OP_SETTABLE, pc.tableReg, pc.keyReg, valBase+i, 0), storeLine)
			} else if pc.fieldK >= 0 {
				fs.emitSetField(pc.tableReg, pc.fieldK, valBase+i, storeLine)
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
		// Inlined `<const>` local in any enclosing scope — error before
		// any other resolution so the message matches a regular const.
		if _, ok := lookupInlinedAny(fs, t.Name); ok {
			c.error(target, errAssignToConst, t.Name)
			return
		}
		// Local?
		if reg, ok := fs.lookupLocal(t.Name); ok {
			if fs.isConst(t.Name) {
				c.error(target, errAssignToConst, t.Name)
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
				c.error(target, errAssignToConst, t.Name)
				return
			}
			tempReg := fs.reserveReg()
			c.compileExprToReg(value, tempReg)
			fs.emit(ABC(OP_SETUPVAL, tempReg, idx, 0, 0), line)
			fs.freeReg = tempReg
			return
		}
		// Local _ENV: _ENV[name] via SETFIELD on local
		if envReg, ok := fs.lookupLocal(envUpvalueName); ok {
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
		// Reference Lua references the live register of a bare-local table
		// operand in the store instruction (luaK_indexed does not copy a
		// local), so an RHS reassignment of that local is observed by the
		// store. Mirror that by using the local register directly; otherwise
		// snapshot the operand into a temp before the RHS.
		// The store is emitted at the target field's line (reference's current
		// line after parsing the indexed target), not the statement line — so a
		// runtime "index a nil value" on the target reports the field's line.
		storeLine := exprEndLine(target)
		startReg := fs.freeReg
		tableReg := c.indexAssignOperand(t.Table, nil)
		valReg := fs.reserveReg()
		c.compileExprToReg(value, valReg)
		fieldK := fs.stringConstant(t.Field)
		fs.emitSetField(tableReg, fieldK, valReg, storeLine)
		fs.freeReg = startReg

	case *ast.IndexExpr:
		storeLine := exprEndLine(target)
		startReg := fs.freeReg
		tableReg := c.indexAssignOperand(t.Table, nil)
		if n, ok := t.Key.(*ast.NumberExpr); ok && n.Value >= 0 && n.Value <= int64(MaxArgC) {
			valReg := fs.reserveReg()
			c.compileExprToReg(value, valReg)
			fs.emit(ABC(OP_SETI, tableReg, int(n.Value), valReg, 0), storeLine)
		} else {
			keyReg := c.indexAssignOperand(t.Key, nil)
			valReg := fs.reserveReg()
			c.compileExprToReg(value, valReg)
			fs.emit(ABC(OP_SETTABLE, tableReg, keyReg, valReg, 0), storeLine)
		}
		fs.freeReg = startReg

	default:
		c.error(target, "invalid assignment target")
	}
}

// indexAssignOperand returns the register to use for the table or key operand
// of an indexed/field assignment target. If the operand is a bare local
// variable, its live register is returned directly (no copy) — matching
// reference Lua, where the store instruction references the local's register so
// a reassignment of that local while evaluating the RHS is observed by the
// store. Any other operand is compiled into a freshly reserved temp (pinned
// before the RHS, as reference does for non-local operands).
//
// For multiple assignment, conflictsLater reports whether a later target in the
// same statement reassigns the given local register; if so the operand is
// snapshotted into a temp to preserve its original value, mirroring reference
// Lua's check_conflict(). It is nil for single assignment (no conflict
// possible).
func (c *compiler) indexAssignOperand(e ast.Expr, conflictsLater func(reg int) bool) int {
	fs := c.fs
	if ne, ok := e.(*ast.NameExpr); ok {
		// Inlined consts are scalar literals, not live locals — fall through
		// to load them into a temp.
		if _, inlined := lookupInlinedAny(fs, ne.Name); !inlined {
			if reg, ok := fs.lookupLocal(ne.Name); ok {
				if conflictsLater == nil || !conflictsLater(reg) {
					return reg
				}
			}
		}
	}
	reg := fs.reserveReg()
	c.compileExprToReg(e, reg)
	return reg
}

// assignToTarget stores the value in srcReg into an assignment target
// (local, upvalue, global, field, or index).
func (c *compiler) assignToTarget(target ast.Expr, srcReg int, line int) {
	fs := c.fs

	switch t := target.(type) {
	case *ast.NameExpr:
		// Inlined `<const>` local in any enclosing scope — error first.
		if _, ok := lookupInlinedAny(fs, t.Name); ok {
			c.error(target, errAssignToConst, t.Name)
			return
		}
		if reg, ok := fs.lookupLocal(t.Name); ok {
			if fs.isConst(t.Name) {
				c.error(target, errAssignToConst, t.Name)
				return
			}
			if reg != srcReg {
				fs.emit(ABC(OP_MOVE, reg, srcReg, 0, 0), line)
			}
			return
		}
		if idx, ok := c.resolveUpvalue(fs, t.Name); ok {
			if c.isConstUpvalue(fs, t.Name) {
				c.error(target, errAssignToConst, t.Name)
				return
			}
			fs.emit(ABC(OP_SETUPVAL, srcReg, idx, 0, 0), line)
			return
		}
		// Local _ENV: _ENV[name] via SETFIELD on local
		if envReg, ok := fs.lookupLocal(envUpvalueName); ok {
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
	// In Lua 5.4's one-pass compiler, the store instruction (SETTABUP)
	// gets the parser's current line after the full RHS is parsed. For
	// multi-line expressions, this is the last line of the expression,
	// not the assignment target's line.
	storeLine := exprEndLine(value)
	if envReg, ok := fs.lookupLocal(envUpvalueName); ok {
		fs.emitSetField(envReg, nameK, tempReg, storeLine)
	} else {
		envUV := c.resolveEnv()
		fs.emitSetTabUp(envUV, nameK, tempReg, storeLine)
	}
	fs.freeReg = tempReg
}

// ---------------------------------------------------------------------------
// Expression statements (function calls)
// ---------------------------------------------------------------------------

// compileExprStmt compiles a bare expression statement (must be a function call).
func (c *compiler) compileExprStmt(s *ast.ExprStmt) {
	fs := c.fs

	switch e := s.Expr.(type) {
	case *ast.FuncCallExpr:
		base := fs.freeReg
		c.compileFuncCall(e, base, 1, e.P.Line) // 1 = discard results (C=1)
	case *ast.MethodCallExpr:
		base := fs.freeReg
		c.compileMethodCall(e, base, 1, e.P.Line) // 1 = discard results
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
				// Use the call's own line (not the 'return' line) so a runtime
				// call error reports the line of the call, matching reference
				// Lua and golua's non-tail call paths.
				c.compileFuncCall(call, base, 0, call.P.Line) // compile the call
				// Replace the CALL with TAILCALL
				lastPC := fs.pc() - 1
				inst := fs.proto.Code[lastPC]
				if inst.OpCode() == OP_CALL {
					fs.proto.Code[lastPC] = ABC(OP_TAILCALL, inst.A(), inst.B(), 0, 0)
				}
				return
			} else if call, ok := s.Values[0].(*ast.MethodCallExpr); ok {
				base := fs.freeReg
				// Use the call's own line (see the FuncCall case above).
				c.compileMethodCall(call, base, 0, call.P.Line) // compile the method call
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
				c.error(nil, errControlStructureTooLong)
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
				c.error(nil, errControlStructureTooLong)
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

	// Create closure in a temp register — emit CLOSURE on the 'end' line
	closureLine := s.Func.EndLine
	if closureLine == 0 {
		closureLine = line
	}
	reg := fs.reserveReg()
	fs.emit(ABx(OP_CLOSURE, reg, protoIdx), closureLine)

	// Assign to the target
	switch name := s.Name.(type) {
	case *ast.NameExpr:
		// Inlined `<const>` local — assignment is an error.
		if _, ok := lookupInlinedAny(fs, name.Name); ok {
			c.error(s.Name, errAssignToConst, name.Name)
			fs.freeReg = reg
			return
		}
		// Simple name: could be local, upvalue, or global
		if localReg, ok := fs.lookupLocal(name.Name); ok {
			if fs.isConst(name.Name) {
				c.error(s.Name, errAssignToConst, name.Name)
				fs.freeReg = reg
				return
			}
			fs.emit(ABC(OP_MOVE, localReg, reg, 0, 0), line)
		} else if uvIdx, ok := c.resolveUpvalue(fs, name.Name); ok {
			if c.isConstUpvalue(fs, name.Name) {
				c.error(s.Name, errAssignToConst, name.Name)
				fs.freeReg = reg
				return
			}
			fs.emit(ABC(OP_SETUPVAL, reg, uvIdx, 0, 0), line)
		} else if envReg, ok := fs.lookupLocal(envUpvalueName); ok {
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
	// Emit CLOSURE on the 'end' line, matching Lua 5.4 which emits CLOSURE
	// after parsing 'end', so ls->lastline is the end keyword's line.
	closureLine := s.Func.EndLine
	if closureLine == 0 {
		closureLine = line
	}
	fs.emit(ABx(OP_CLOSURE, reg, protoIdx), closureLine)
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
			if envReg, ok := fs.lookupLocal(envUpvalueName); ok {
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
	closureLine := s.Func.EndLine
	if closureLine == 0 {
		closureLine = line
	}
	reg := fs.reserveReg()
	fs.emit(ABx(OP_CLOSURE, reg, protoIdx), closureLine)

	nameK := fs.stringConstant(s.Name.Name)
	if envReg, ok := fs.lookupLocal(envUpvalueName); ok {
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

	// Vararg prep — use the first body line (not the definition line)
	// to match Lua 5.4, which assigns VARARGPREP to the first line of
	// the function body so it doesn't appear in activelines for the
	// definition line.
	if fe.VarArg {
		varargLine := line
		if fe.Body != nil && len(fe.Body.Stmts) > 0 {
			varargLine = fe.Body.Stmts[0].Pos().Line
		}
		fs.emit(ABC(OP_VARARGPREP, fs.proto.NumParams, 0, 0, 0), varargLine)
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

	fs.closeLine = lastLine
	c.leaveScope(lastLine)

	proto := c.closeFuncState()
	return parentFS.addProto(proto)
}
