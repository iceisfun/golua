package compiler

import "github.com/iceisfun/golua/v2/ast"

// This file holds control-flow statement lowering: if/elseif/else, the loop
// forms (while, repeat, numeric for, generic for), do-blocks, break, and the
// conditional-jump helpers they share. Split out of compile_stmt.go.

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
			eCondLine := c.compileCondJump(elif.Cond, false, eline)
			elifJump := fs.emitJump(eCondLine)
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
			eCondLine := c.compileCondJump(elif.Cond, false, eline)
			elifJump := fs.emitJump(eCondLine)
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
	thenLine := s.ThenLine
	condLine := c.compileCondJump(s.Cond, false, thenLine)
	thenJump := fs.emitJump(condLine) // jump past then-block if false
	fs.enterScope(false)
	c.compileBlock(s.Then)
	c.leaveScope(thenLine)

	// Jump past all else/elseif blocks after then
	if len(s.ElseIfs) > 0 || s.Else != nil {
		// Use the line of the last emitted instruction (end of then-block)
		exitLine := c.lastEmittedLine()
		exitJumps = append(exitJumps, fs.emitJump(exitLine))
	}

	c.patchJump(thenJump)

	// elseif branches
	for _, elif := range s.ElseIfs {
		eiThenLine := elif.ThenLine
		eiCondLine := c.compileCondJump(elif.Cond, false, eiThenLine)
		elifJump := fs.emitJump(eiCondLine)
		fs.enterScope(false)
		c.compileBlock(elif.Then)
		c.leaveScope(eiThenLine)
		exitLine := c.lastEmittedLine()
		exitJumps = append(exitJumps, fs.emitJump(exitLine))
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
//
// The returned line is the source line that the emitted TEST is attributed
// to. Reference Lua attributes the condition's test/jump to the line of the
// *last token of the condition expression*, not the line of the trailing
// `then`/`do` keyword (which is what `fallbackLine` carries). Callers should
// use the returned line for the JMP they emit immediately after, so the
// whole condition test forms a single contiguous line run (matching the
// line-hook trace produced by reference Lua).
func (c *compiler) compileCondJump(cond ast.Expr, jumpOnFalse bool, fallbackLine int) int {
	fs := c.fs

	// Fused condition forms. Each emits a single instruction that already
	// uses OP_TEST's "skip the next instruction if the result ~= k"
	// convention, so the caller's JMP stays exactly where it was and no
	// boolean is materialized (reference Lua's luaK_goiftrue/goiffalse).
	switch e := cond.(type) {
	case *ast.ParenExpr:
		// Parentheses are transparent for a condition: only the truthiness
		// of a single value matters, and every path below (compare, TEST)
		// consumes exactly one value.
		return c.compileCondJump(e.Inner, jumpOnFalse, fallbackLine)
	case *ast.UnopExpr:
		// `not x` flips the jump sense instead of materializing NOT + TEST,
		// matching reference Lua's luaK_goiftrue/goiffalse negation.
		if e.Op == "not" {
			return c.compileCondJump(e.Operand, !jumpOnFalse, fallbackLine)
		}
	case *ast.BinopExpr:
		// Top-level comparison: fuse into a single compare instruction that
		// shares OP_TEST's skip convention, matching reference Lua's
		// LT/LE/EQ (+ immediate forms) + JMP shape instead of materializing
		// a boolean and testing it.
		if isComparisonOp(e.Op) {
			line := exprEndLine(e)
			c.compileComparisonTest(e, jumpOnFalse, line)
			return line
		}
	case *ast.NameExpr:
		// A plain in-register local is tested in place — TEST only reads the
		// register, so a live local is safe and the MOVE to a temp is
		// unnecessary (matches reference Lua's exp2anyreg on VLOCAL).
		if reg, ok := plainLocalReg(fs, cond); ok {
			line := e.P.Line
			k := 1
			if !jumpOnFalse {
				k = 0
			}
			fs.emit(ABC(OP_TEST, reg, 0, 0, k), line)
			return line
		}
	}

	reg := fs.freeReg
	c.compileExprToReg(cond, reg)
	// Use the line of the last instruction emitted while materializing the
	// condition. This is the condition's own end line, matching reference
	// Lua, rather than the keyword (`then`/`do`) line. If the condition
	// emitted no instruction (e.g. an empty proto), fall back to the keyword.
	line := fallbackLine
	if l := c.lastEmittedLine(); l > 0 {
		line = l
	}
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
	return line
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
		if fs.needsClose(scope.nLocals) {
			// __close fires here; attribute to the last body statement
			// (ls->lastline in reference Lua), not the 'while' line.
			fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), c.fs.blockLastLine)
		}

		// Jump back to loop start (unconditional) — use last emitted line
		backLine := c.lastEmittedLine()
		backJump := fs.emitJump(backLine)
		offset := loopStart - (fs.pc()) // negative
		if offset > MaxSJ || offset < MinSJ {
			c.error(nil, errControlStructureTooLong)
		} else {
			fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)
		}

		c.leaveScope(line)
		return
	}

	loopStart := fs.pc()
	fs.enterScope(true)

	// Test condition
	condLine := c.compileCondJump(s.Cond, false, line)
	exitJump := fs.emitJump(condLine)

	// Body
	c.compileBlock(s.Body)

	// Close upvalues for body locals before jumping back.
	// This ensures each iteration gets its own closed upvalue copy.
	scope := fs.scopes[len(fs.scopes)-1]
	if fs.needsClose(scope.nLocals) {
		// __close fires here; attribute to the last body statement
		// (ls->lastline in reference Lua), not the 'while' line.
		fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), c.fs.blockLastLine)
	}

	// Jump back to condition — use the last emitted line (typically the
	// last line of the loop body), matching Lua 5.4 which uses ls->lastline
	// (the 'end' keyword line) for the backward JMP.
	backLine := c.lastEmittedLine()
	backJump := fs.emitJump(backLine)
	offset := loopStart - (fs.pc()) // negative
	if offset > MaxSJ || offset < MinSJ {
		c.error(nil, errControlStructureTooLong)
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

	// Constant conditions can skip the generic boolean materialization path,
	// which keeps repeat/until instruction shape much closer to Lua 5.4.
	if isConstTrue(s.Cond) {
		// The loop exits after one pass; __close fires at scope exit, which
		// reference Lua attributes to the until-condition line (ls->lastline
		// after the condition is consumed), not the body's last statement.
		// Emit the firing OP_CLOSE explicitly at that line, mirroring the
		// generic path; leaveScope's cleanup close is a runtime no-op on the
		// already-closed TBC slot.
		scope := fs.scopes[len(fs.scopes)-1]
		if fs.needsClose(scope.nLocals) {
			fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), s.Cond.Pos().Line)
		}
		c.leaveScope(line)
		return
	}
	if isConstFalsy(s.Cond) {
		scope := fs.scopes[len(fs.scopes)-1]
		if fs.needsClose(scope.nLocals) {
			fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), s.Cond.Pos().Line)
		}
		backJump := fs.emitJump(c.lastEmittedLine())
		offset := loopStart - fs.pc()
		if offset > MaxSJ || offset < MinSJ {
			c.error(nil, errControlStructureTooLong)
		} else {
			fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)
		}
		c.leaveScope(line)
		return
	}

	// Evaluate condition (may reference body locals; the fused forms read
	// operand registers in place, which is safe — nothing below clears them).
	// The JMP emitted after the test fires when the condition is falsy
	// (keep looping) and falls through when it is truthy (exit).
	condLine := c.compileCondJump(s.Cond, false, s.Cond.Pos().Line)
	backJump := fs.emitJump(condLine)

	// Close upvalues for body locals before repeating, so each iteration gets
	// its own closed upvalue copy. The test and its JMP must stay adjacent, so
	// — like reference Lua (lparser.c repeatstat) — the OP_CLOSE lives in a
	// stub on the repeat path: the truthy fall-through jumps over it, and the
	// exit path closes just past it. needsClose is queried only now because
	// compiling the condition itself can capture a body local
	// (`until (function() return x end)()`).
	scope := fs.scopes[len(fs.scopes)-1]
	if fs.needsClose(scope.nLocals) {
		exitJump := fs.emitJump(condLine) // truthy fall-through: skip the stub
		c.patchJump(backJump)             // falsy path lands on the stub
		fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), condLine)
		backJump = fs.emitJump(condLine) // ... then jump back
		c.patchJump(exitJump)
		// Exit path: fire __close here, attributed to the until-condition
		// line as before. leaveScope's cleanup close below is a runtime
		// no-op on the already-closed slot.
		fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), condLine)
	}

	offset := loopStart - (backJump + 1)
	if offset > MaxSJ || offset < MinSJ {
		c.error(nil, errControlStructureTooLong)
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
// Lua 5.5 layout: 2 internal control registers plus the visible loop variable,
// which doubles as the control register (no separate safe-copy slot as in 5.4).
// FORPREP rewrites the init/limit/step inputs in place so that after prep
// base   = loop counter (int) or limit (float),
// base+1 = step,
// base+2 = control variable (the visible 'i').
func (c *compiler) compileForNumStmt(s *ast.ForNumStmt) {
	fs := c.fs
	line := s.P.Line

	fs.enterScope(true)

	// Reserve 3 registers: init, limit, step are compiled into base, base+1,
	// base+2; FORPREP rewrites them in place, leaving the visible 'i' at base+2.
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

	// FORPREP — jumps to FORLOOP if not to run. Reference Lua attributes the
	// instruction (and thus any "bad 'for' limit/initial value/step" or
	// "'for' step is zero" runtime error) to the line of the 'do' keyword, not
	// the 'for' keyword, since luaK_code stamps it with lastline after 'do' is
	// consumed.
	prepLine := s.DoLine
	if prepLine == 0 {
		prepLine = line
	}
	forPrepPC := fs.emit(ABx(OP_FORPREP, base, 0), prepLine)

	// Body-live registers: 2 control slots (base, base+1) + visible i (base+2).
	fs.freeReg = base + 3
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	fs.checkRegLimit()

	// Add the 2 internal control registers as hidden locals to protect their
	// registers so freeReg won't be reset below base+3 during the loop body.
	//
	// Lua 5.5 checks the MAXVARS limit in two phases (lparser.c fornum/forbody):
	// the 2 internal "(for state)" vars are adjustlocalvars'd before 'do', and
	// the visible control variable inside forbody, after 'do' is consumed. So
	// the near-token is 'do' if the internals alone overflow, otherwise the
	// token following 'do' (first body statement, or 'end' for an empty body).
	bodyLine, bodyNear := c.forBodyNextInfo(s.Body, s.EndLine)
	fs.checkVarLimitAt(2, line, "do")
	fs.checkVarLimitAt(3, bodyLine, bodyNear)
	fs.locals = append(fs.locals,
		localVar{name: forStateVarName, reg: base, startPC: fs.pc()},
		localVar{name: forStateVarName, reg: base + 1, startPC: fs.pc()},
	)
	fs.nActVar += 2

	// The control register at base+2 is the visible loop variable, exposed as a
	// const local (Lua 5.5: for-loop control variables are read-only).
	fs.locals = append(fs.locals, localVar{
		name:    s.Name.Name,
		reg:     base + 2,
		startPC: fs.pc(),
		attrib:  attribConst,
	})
	fs.nActVar++
	// The 2 for-state slots and the loop variable stay live across a
	// body-end label (see scopeInfo.headerLocals).
	fs.scopes[len(fs.scopes)-1].headerLocals = 3

	// Body
	c.compileBlock(s.Body)

	// Close upvalues at the loop variable (base+2) and above before looping back,
	// but only if any variable from the loop variable onwards was captured by an
	// inner closure or has a <close> attribute. Skipping CLOSE when not needed
	// avoids inflating instruction counts (which affects debug count hooks).
	needClose := false
	for i := len(fs.locals) - 1; i >= 0; i-- {
		// Inlined <const> locals consume no register (reg = -1); they must be
		// skipped rather than terminating the scan, otherwise a captured loop
		// variable or a <close> local appended below one is never seen and the
		// per-iteration OP_CLOSE is wrongly omitted.
		if fs.locals[i].inlined {
			continue
		}
		if fs.locals[i].reg < base+2 {
			break
		}
		if fs.locals[i].captured || fs.locals[i].attrib == attribClose {
			needClose = true
			break
		}
	}
	if needClose {
		// __close fires here; attribute to the last body statement
		// (ls->lastline in reference Lua), not the 'for' line.
		fs.emit(ABC(OP_CLOSE, base+2, 0, 0, 0), c.fs.blockLastLine)
	}

	// FORLOOP — jumps back to just after FORPREP
	loopPC := fs.emit(ABx(OP_FORLOOP, base, 0), line)

	// Patch FORPREP to jump to FORLOOP
	bodyLen := loopPC - forPrepPC - 1
	if bodyLen > MaxArgBx {
		c.error(nil, errControlStructureTooLong)
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
// Lua 5.5 layout: 3 internal control registers (iterator function, state,
// closing variable) plus the loop variables, the first of which doubles as the
// control register. The iterator explist still fills 4 slots (function, state,
// control, closing); TFORPREP swaps the control/closing values so the closing
// variable lands in the internal slot at base+2 and the control variable lands
// at base+3 (the user's first loop variable).
func (c *compiler) compileForInStmt(s *ast.ForInStmt) {
	fs := c.fs
	line := s.P.Line

	fs.enterScope(true)

	// Reserve 4 explist slots: iterator, state, control, closing
	base := fs.freeReg

	// Compile iterator expressions into base, base+1, base+2, base+3
	// The 4th value (base+3) is the closing variable; TFORPREP later swaps it
	// down to base+2 (the internal to-be-closed slot).
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
				isLast := i == nIter-1
				if isLast && isMultiRet(iter) {
					// Last expression is multires — expand to fill remaining control slots.
					remaining := 4 - i
					fs.freeReg = base + i
					c.compileExprMultiRet(iter, remaining)
					fs.freeReg = base + 4
					if fs.freeReg > fs.maxReg {
						fs.maxReg = fs.freeReg
					}
				} else {
					c.compileExprToReg(iter, base+i)
					fs.freeReg = base + i + 1
					if fs.freeReg > fs.maxReg {
						fs.maxReg = fs.freeReg
					}
				}
			} else {
				// Lua 5.4 still evaluates extra explist expressions for side
				// effects even though only the first four results seed the loop.
				tmp := fs.freeReg
				c.compileExprToReg(iter, tmp)
				fs.freeReg = tmp
			}
		}
		// Fill missing with nil (only needed when last expr was not multires)
		lastFillsRemaining := nIter > 0 && nIter <= 4 && isMultiRet(s.Iters[nIter-1])
		for i := nIter; i < 4 && !lastFillsRemaining; i++ {
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

	// Add the 3 internal for-in control registers as hidden locals (Lua 5.5:
	// iterator function, state, closing variable). The control variable is
	// folded into the user's first loop variable at base+3 (no dedicated
	// control slot as in Lua 5.4). TFORPREP swaps the control/closing values
	// into place at runtime and marks base+2 as to-be-closed, so there is no
	// separate OP_TBC instruction.
	localStartPC := fs.pc()
	// Lua 5.5 checks the MAXVARS limit in two phases (lparser.c forlist/forbody):
	// the 3 internal "(for state)" vars are adjustlocalvars'd after 'in <explist>'
	// but before 'do', and the user loop variables inside forbody, after 'do'.
	// So the near-token is 'do' if the internals alone overflow, otherwise the
	// token following 'do' (first body statement, or 'end' for an empty body).
	bodyLine, bodyNear := c.forBodyNextInfo(s.Body, s.EndLine)
	fs.checkVarLimitAt(3, line, "do")
	fs.checkVarLimitAt(3+len(s.Names), bodyLine, bodyNear)
	fs.locals = append(fs.locals,
		localVar{name: forStateVarName, reg: base, startPC: localStartPC},
		localVar{name: forStateVarName, reg: base + 1, startPC: localStartPC},
		// base+2 is the closing variable (after the TFORPREP swap); mark it
		// <close> so leaveScope emits the loop-end OP_CLOSE that fires __close.
		localVar{name: forStateVarName, reg: base + 2, startPC: localStartPC, attrib: attribClose},
	)
	fs.nActVar += 3

	// TFORPREP
	tforPrepPC := fs.emit(ABx(OP_TFORPREP, base, 0), line)

	// Loop variables start at base+3 (the first is the control variable).
	// Record their index in fs.locals now, before the body adds more locals:
	// computing it after compileBlock would include body locals that are still
	// active at the fixup point (leaveScope removes them later), which would
	// make the visibility fixup below grab the wrong entries.
	nVars := len(s.Names)
	loopVarStart := len(fs.locals)
	for i, name := range s.Names {
		reg := base + 3 + i
		fs.freeReg = reg + 1
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
		// Lua 5.5: the first loop variable (control variable) is read-only
		attrib := ""
		if i == 0 {
			attrib = attribConst
		}
		fs.locals = append(fs.locals, localVar{
			name:    name.Name,
			reg:     reg,
			startPC: fs.pc(),
			attrib:  attrib,
		})
		fs.nActVar++
	}
	fs.checkRegLimit()
	// The 3 for-state slots and the loop variables stay live across a
	// body-end label (see scopeInfo.headerLocals).
	fs.scopes[len(fs.scopes)-1].headerLocals = 3 + nVars

	// Body
	c.compileBlock(s.Body)

	// Close upvalues at the loop variables (base+3) and above before next
	// iteration. This ensures each iteration gets its own closed upvalue copy.
	// The closing variable at base+2 stays open across iterations (it is closed
	// once when the loop scope exits).
	scope := fs.scopes[len(fs.scopes)-1]
	if fs.needsClose(scope.nLocals + 3) {
		// __close of body locals fires here; attribute to the last body
		// statement (ls->lastline in reference Lua), not the 'for' line.
		fs.emit(ABC(OP_CLOSE, base+3, 0, 0, 0), c.fs.blockLastLine)
	}

	// TFORCALL — calls the iterator. Reference (lparser.c forlist) captures the
	// line right after consuming 'in', i.e. the FIRST token of the iterator
	// explist, and fixlines OP_TFORCALL/OP_TFORLOOP to it — so an error raised
	// inside the iterator reports the explist start line.
	iterLine := line
	if len(s.Iters) > 0 {
		iterLine = s.Iters[0].Pos().Line
	}
	tforCallPC := fs.emit(ABC(OP_TFORCALL, base, 0, nVars, 0), iterLine)
	_ = tforCallPC

	// TFORLOOP — checks if control variable is nil
	tforLoopPC := fs.emit(ABx(OP_TFORLOOP, base, 0), line)

	// Patch TFORPREP to jump to TFORCALL
	bodyLen := tforCallPC - tforPrepPC - 1
	if bodyLen > MaxArgBx {
		c.error(nil, errControlStructureTooLong)
	}
	fs.proto.Code[tforPrepPC] = fs.proto.Code[tforPrepPC].SetBx(bodyLen)

	// Patch TFORLOOP to jump back to loop body (after TFORPREP)
	backLen := tforLoopPC - tforPrepPC - 1
	if backLen > MaxArgBx {
		c.error(nil, errControlStructureTooLong)
	}
	fs.proto.Code[tforLoopPC] = fs.proto.Code[tforLoopPC].SetBx(backLen)

	// Fix loop variable visibility: they should NOT be visible during the
	// TFORCALL iterator invocation (matching Lua 5.4 where loop vars start
	// after TFORCALL). Emit separate debug entries for the body range only.
	// loopVarStart was captured before the body was compiled (see above).
	bodyEnd := tforCallPC // exclusive: body ends just before TFORCALL
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
		// Lua 5.5: error is anchored at the break statement's own line and
		// uses "near 'break'" wording (5.4 used "at line N" at the EOF line).
		c.error(s, "break outside loop near 'break'")
		return
	}
	// Reference Lua compiles break as a goto to the loop block's break label,
	// which is created past that block's own OP_CLOSE — so a break that leaves
	// the scope of a captured or to-be-closed variable must close by itself.
	// Whether it does is only known when the loop block ends: a capture can sit
	// textually after the break and still run before it. Emit the close now
	// only when the answer is already yes; otherwise leave a placeholder for
	// leaveScope to activate.
	closed := fs.needsClose(scope.nLocals)
	if closed {
		fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), s.P.Line)
	}
	jpc := fs.emitJump(s.P.Line)
	e := c.recordJumpClose(jpc, -1, scope.nLocals, scope.baseReg, s.P.Line, closed)
	scope.breakJumps = append(scope.breakJumps, e)
}
