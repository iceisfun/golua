package compiler

import (
	"fmt"

	"github.com/iceisfun/golua/ast"
)

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Compile transforms a parsed AST block into a top-level function prototype.
// The source parameter names the chunk for error messages and debug info.
// The returned [Proto] represents a vararg function with a single upvalue (_ENV)
// that, when executed, runs the chunk's statements.
//
// Compile may return an error for programs that exceed compiler limits
// (register count, local variable count, upvalue count) or contain
// unresolved goto labels.
func Compile(source string, block *ast.Block, opts ...CompileOption) (*Proto, error) {
	cfg := compileConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	c := &compiler{limits: cfg.limits.effective()}
	p := c.compileChunk(source, block)
	if c.err != nil {
		return nil, c.err
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// exprResult tracks where an expression's value ended up
// ---------------------------------------------------------------------------

type exprKind int

const (
	exprReg     exprKind = iota // value is in register info
	exprConst                   // value is constant index info
	exprTrue                    // literal true
	exprFalse                   // literal false
	exprNil                     // literal nil
	exprJump                    // conditional jump — info is index into pending jumps
	exprRelocate                // instruction needs A set — info is pc
	exprCall                    // function call — info is pc of CALL instruction
	exprVarArg                  // vararg — info is pc of VARARG instruction
)

type exprResult struct {
	kind exprKind
	info int // meaning depends on kind
	t    int // patch list for jumps-when-true  (-1 = no pending)
	f    int // patch list for jumps-when-false (-1 = no pending)
}

func (e exprResult) hasJumps() bool { return e.t != -1 || e.f != -1 }

const noJump = -1

// ---------------------------------------------------------------------------
// funcState — per-function compiler state
// ---------------------------------------------------------------------------

type localVar struct {
	name     string
	reg      int
	startPC  int
	attrib   string // "", "const", "close"
	captured bool   // true if captured as an upvalue by an inner closure
}

type scopeInfo struct {
	nLocals    int  // number of locals when scope opened
	breakList  int  // patch list head for break jumps (-1 = none)
	isLoop     bool // is this a loop scope?
	firstLabel int  // index into labels slice
}

type labelInfo struct {
	name    string
	pc      int
	line    int
	nLocals int // number of active locals when label was defined
}

type pendingGoto struct {
	name    string
	pc      int // jump instruction pc
	nLocals int // number of locals at goto
	line    int
	closePC int // pc of placeholder OP_CLOSE (-1 if none)
}

type funcState struct {
	c       *compiler // backpointer to compiler (for limits and error reporting)
	proto   *Proto
	parent  *funcState
	freeReg int
	maxReg  int // high-water mark for register allocation
	nActVar int // number of active local variables

	locals      []localVar
	scopes      []scopeInfo
	labels      []labelInfo
	pendGotos   []pendingGoto
	upvalues    []UpvalDesc
	upvalLookup map[string]int // name → index in upvalues
}

// ---------------------------------------------------------------------------
// compiler — top-level state
// ---------------------------------------------------------------------------

type compiler struct {
	fs     *funcState
	err    error
	limits CompilerLimits
}

func (c *compiler) error(pos interface{}, format string, args ...interface{}) {
	if c.err == nil {
		c.err = fmt.Errorf(format, args...)
	}
}

// ---------------------------------------------------------------------------
// Function state helpers
// ---------------------------------------------------------------------------

func (c *compiler) newFuncState(source string, parent *funcState) *funcState {
	fs := &funcState{
		c: c,
		proto: &Proto{
			Source: source,
		},
		parent:      parent,
		upvalLookup: make(map[string]int),
	}
	c.fs = fs
	return fs
}

func (c *compiler) closeFuncState() *Proto {
	fs := c.fs
	p := fs.proto

	// Check for unresolved gotos (label not visible)
	for _, pg := range fs.pendGotos {
		c.error(nil, "no visible label '%s' for <goto> at line %d", pg.name, pg.line)
	}

	// Close all remaining locals
	for i := range fs.locals {
		if fs.locals[i].startPC >= 0 {
			p.Locals = append(p.Locals, LocalVar{
				Name:    fs.locals[i].name,
				StartPC: fs.locals[i].startPC,
				EndPC:   len(p.Code),
			})
		}
	}

	p.MaxStack = fs.maxReg
	p.Upvalues = fs.upvalues
	c.fs = fs.parent
	return p
}

// regTop returns the first register past all active local variables.
// When locals occupy contiguous registers from R(0), this equals nActVar.
// When condition temporaries create gaps (e.g., while/for loop conditions
// allocate a temp register before body locals are declared), this may be
// higher than nActVar.
//
// This is a correctness guard: temporary registers used for expression
// evaluation must not reuse registers assigned to user-visible locals.
// Using regTop() instead of nActVar to reset freeReg prevents a class
// of compiler bugs where condition temporaries push local variables to
// higher registers than nActVar accounts for.
func (fs *funcState) regTop() int {
	top := 0
	start := len(fs.locals) - fs.nActVar
	for i := start; i < len(fs.locals); i++ {
		if r := fs.locals[i].reg + 1; r > top {
			top = r
		}
	}
	return top
}

// maxReg is computed from the high-water mark.
// We track it on funcState.
func (fs *funcState) reserveReg() int {
	r := fs.freeReg
	fs.freeReg++
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	if fs.freeReg > fs.c.limits.MaxRegs {
		fs.c.error(nil, "too many registers (limit is %d)", fs.c.limits.MaxRegs)
	}
	return r
}

func (fs *funcState) reserveRegs(n int) int {
	base := fs.freeReg
	fs.freeReg += n
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	if fs.freeReg > fs.c.limits.MaxRegs {
		fs.c.error(nil, "too many registers (limit is %d)", fs.c.limits.MaxRegs)
	}
	return base
}

func (fs *funcState) emit(inst Instruction, line int) int {
	pc := len(fs.proto.Code)
	fs.proto.Code = append(fs.proto.Code, inst)
	fs.proto.Lines = append(fs.proto.Lines, line)
	return pc
}

func (fs *funcState) pc() int {
	return len(fs.proto.Code)
}

// emitGetTabUp emits OP_GETTABUP or, when kIdx > MaxArgC, a fallback
// sequence (GETUPVAL + LOADK + GETTABLE) that avoids 8-bit overflow.
func (fs *funcState) emitGetTabUp(reg, upIdx, kIdx int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_GETTABUP, reg, upIdx, kIdx, 0), line)
		return
	}
	saved := fs.freeReg
	tmpEnv := fs.reserveReg()
	tmpKey := fs.reserveReg()
	fs.emit(ABC(OP_GETUPVAL, tmpEnv, upIdx, 0, 0), line)
	fs.emit(ABx(OP_LOADK, tmpKey, kIdx), line)
	fs.emit(ABC(OP_GETTABLE, reg, tmpEnv, tmpKey, 0), line)
	fs.freeReg = saved
}

// emitGetField emits OP_GETFIELD or, when kIdx > MaxArgC, a fallback
// sequence (LOADK + GETTABLE).
func (fs *funcState) emitGetField(reg, tableReg, kIdx int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_GETFIELD, reg, tableReg, kIdx, 0), line)
		return
	}
	saved := fs.freeReg
	tmpKey := fs.reserveReg()
	fs.emit(ABx(OP_LOADK, tmpKey, kIdx), line)
	fs.emit(ABC(OP_GETTABLE, reg, tableReg, tmpKey, 0), line)
	fs.freeReg = saved
}

// emitSetTabUp emits OP_SETTABUP or, when kIdx > MaxArgC, a fallback
// sequence (GETUPVAL + LOADK + SETTABLE).
func (fs *funcState) emitSetTabUp(upIdx, kIdx, valReg int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_SETTABUP, upIdx, kIdx, valReg, 0), line)
		return
	}
	saved := fs.freeReg
	tmpEnv := fs.reserveReg()
	tmpKey := fs.reserveReg()
	fs.emit(ABC(OP_GETUPVAL, tmpEnv, upIdx, 0, 0), line)
	fs.emit(ABx(OP_LOADK, tmpKey, kIdx), line)
	fs.emit(ABC(OP_SETTABLE, tmpEnv, tmpKey, valReg, 0), line)
	fs.freeReg = saved
}

// emitSetField emits OP_SETFIELD or, when kIdx > MaxArgC, a fallback
// sequence (LOADK + SETTABLE).
func (fs *funcState) emitSetField(tableReg, kIdx, valReg int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_SETFIELD, tableReg, kIdx, valReg, 0), line)
		return
	}
	saved := fs.freeReg
	tmpKey := fs.reserveReg()
	fs.emit(ABx(OP_LOADK, tmpKey, kIdx), line)
	fs.emit(ABC(OP_SETTABLE, tableReg, tmpKey, valReg, 0), line)
	fs.freeReg = saved
}

// emitSelf emits OP_SELF or, when kIdx > MaxArgC, a fallback
// sequence (MOVE + LOADK + GETTABLE).
func (fs *funcState) emitSelf(base, objReg, kIdx int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_SELF, base, objReg, kIdx, 0), line)
		return
	}
	saved := fs.freeReg
	fs.emit(ABC(OP_MOVE, base+1, objReg, 0, 0), line)
	tmpKey := fs.reserveReg()
	fs.emit(ABx(OP_LOADK, tmpKey, kIdx), line)
	fs.emit(ABC(OP_GETTABLE, base, objReg, tmpKey, 0), line)
	fs.freeReg = saved
}

func (fs *funcState) addConstant(v Value) int {
	// Deduplicate constants
	for i, existing := range fs.proto.Constants {
		if valEqual(existing, v) {
			return i
		}
	}
	idx := len(fs.proto.Constants)
	fs.proto.Constants = append(fs.proto.Constants, v)
	return idx
}

func valEqual(a, b Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case ValNil:
		return true
	case ValFalse, ValTrue:
		return true
	case ValInt:
		return a.IVal == b.IVal
	case ValFloat:
		return a.FVal == b.FVal
	case ValString:
		return a.SVal == b.SVal
	}
	return false
}

func (fs *funcState) addProto(p *Proto) int {
	idx := len(fs.proto.Protos)
	fs.proto.Protos = append(fs.proto.Protos, p)
	return idx
}

func (fs *funcState) stringConstant(s string) int {
	return fs.addConstant(StringValue(s))
}

// ---------------------------------------------------------------------------
// Local variables
// ---------------------------------------------------------------------------

// checkVarLimit checks that adding count new locals won't exceed the limit.
func (fs *funcState) checkVarLimit(count int) {
	if fs.nActVar+count > fs.c.limits.MaxVars {
		fs.c.error(nil, "too many local variables (limit is %d)", fs.c.limits.MaxVars)
	}
}

// checkRegLimit checks that the current freeReg doesn't exceed the register limit.
func (fs *funcState) checkRegLimit() {
	if fs.freeReg > fs.c.limits.MaxRegs {
		fs.c.error(nil, "too many registers (limit is %d)", fs.c.limits.MaxRegs)
	}
}

func (fs *funcState) addLocal(name string, attrib string) int {
	fs.checkVarLimit(1)
	reg := fs.reserveReg()
	fs.locals = append(fs.locals, localVar{
		name:    name,
		reg:     reg,
		startPC: -1, // not yet active
		attrib:  attrib,
	})
	fs.nActVar++
	return reg
}

func (fs *funcState) activateLocal(idx int) {
	if idx < len(fs.locals) {
		fs.locals[idx].startPC = fs.pc()
	}
}

func (fs *funcState) lookupLocal(name string) (int, bool) {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		if fs.locals[i].name == name && fs.locals[i].startPC >= 0 {
			return fs.locals[i].reg, true
		}
	}
	return 0, false
}

func (fs *funcState) needsClose(fromLocal int) bool {
	start := len(fs.locals) - (fs.nActVar - fromLocal)
	if start < 0 {
		start = 0
	}
	for i := start; i < len(fs.locals); i++ {
		if fs.locals[i].attrib == "close" || fs.locals[i].captured {
			return true
		}
	}
	return false
}

func (fs *funcState) isConst(name string) bool {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		if fs.locals[i].name == name && fs.locals[i].startPC >= 0 {
			return fs.locals[i].attrib == "const"
		}
	}
	return false
}

func (c *compiler) isConstUpvalue(fs *funcState, name string) bool {
	if fs.parent == nil {
		return false
	}
	if fs.parent.isConst(name) {
		return true
	}
	return c.isConstUpvalue(fs.parent, name)
}

func (fs *funcState) markCaptured(name string) {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		if fs.locals[i].name == name && fs.locals[i].startPC >= 0 {
			fs.locals[i].captured = true
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Upvalues
// ---------------------------------------------------------------------------

func (fs *funcState) lookupUpvalue(name string) (int, bool) {
	if idx, ok := fs.upvalLookup[name]; ok {
		return idx, true
	}
	return 0, false
}

func (fs *funcState) addUpvalue(name string, inStack bool, index int) int {
	if idx, ok := fs.upvalLookup[name]; ok {
		return idx
	}
	idx := len(fs.upvalues)
	if idx >= fs.c.limits.MaxUpvals {
		fs.c.error(nil, "too many upvalues (limit is %d)", fs.c.limits.MaxUpvals)
		return 0
	}
	fs.upvalues = append(fs.upvalues, UpvalDesc{
		Name:    name,
		InStack: inStack,
		Index:   index,
	})
	fs.upvalLookup[name] = idx
	return idx
}

// resolveUpvalue finds a variable in enclosing functions and creates upvalue
// chain as needed.
func (c *compiler) resolveUpvalue(fs *funcState, name string) (int, bool) {
	// Already an upvalue in this function?
	if idx, ok := fs.lookupUpvalue(name); ok {
		return idx, true
	}
	// No parent? Can't resolve.
	if fs.parent == nil {
		return 0, false
	}
	// Is it a local in the parent?
	if reg, ok := fs.parent.lookupLocal(name); ok {
		fs.parent.markCaptured(name)
		return fs.addUpvalue(name, true, reg), true
	}
	// Is it an upvalue in the parent?
	if idx, ok := c.resolveUpvalue(fs.parent, name); ok {
		return fs.addUpvalue(name, false, idx), true
	}
	return 0, false
}

// ---------------------------------------------------------------------------
// Scope management
// ---------------------------------------------------------------------------

func (fs *funcState) enterScope(isLoop bool) {
	fs.scopes = append(fs.scopes, scopeInfo{
		nLocals:    fs.nActVar,
		breakList:  noJump,
		isLoop:     isLoop,
		firstLabel: len(fs.labels),
	})
}

func (c *compiler) leaveScope(line int) {
	fs := c.fs
	scope := fs.scopes[len(fs.scopes)-1]
	fs.scopes = fs.scopes[:len(fs.scopes)-1]

	// Emit OP_CLOSE if this scope has any to-be-closed or captured variables.
	// This closes upvalues and calls __close metamethods.
	if fs.nActVar > scope.nLocals {
		needClose := false
		start := len(fs.locals) - (fs.nActVar - scope.nLocals)
		for i := start; i < len(fs.locals); i++ {
			if fs.locals[i].attrib == "close" || fs.locals[i].captured {
				needClose = true
				break
			}
		}
		if needClose {
			fs.emit(ABC(OP_CLOSE, scope.nLocals, 0, 0, 0), line)
		}
	}

	// Remove locals from this scope
	for len(fs.locals) > 0 && fs.nActVar > scope.nLocals {
		loc := &fs.locals[len(fs.locals)-1]
		if loc.startPC >= 0 {
			fs.proto.Locals = append(fs.proto.Locals, LocalVar{
				Name:    loc.name,
				StartPC: loc.startPC,
				EndPC:   fs.pc(),
			})
		}
		fs.locals = fs.locals[:len(fs.locals)-1]
		fs.nActVar--
	}

	// Reset freeReg past all remaining locals. We use regTop() instead
	// of scope.nLocals because the remaining locals may occupy registers
	// beyond their count (e.g., when a while loop condition temp creates
	// a gap between outer and inner locals).
	fs.freeReg = fs.regTop()

	// Remove labels from this scope
	fs.labels = fs.labels[:scope.firstLabel]

	// Patch break jumps
	if scope.breakList != noJump {
		c.patchListToHere(scope.breakList)
	}
}

func (fs *funcState) findLoopScope() *scopeInfo {
	for i := len(fs.scopes) - 1; i >= 0; i-- {
		if fs.scopes[i].isLoop {
			return &fs.scopes[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Jump patching
// ---------------------------------------------------------------------------

// emitJump emits a JMP instruction and returns its pc.
func (fs *funcState) emitJump(line int) int {
	return fs.emit(SJ(OP_JMP, 0, 0), line)
}

// patchJump patches the jump at pc to jump to the current pc.
func (c *compiler) patchJump(jpc int) {
	fs := c.fs
	offset := fs.pc() - (jpc + 1) // target - (jpc + 1)
	fs.proto.Code[jpc] = fs.proto.Code[jpc].SetSJ(offset)
}

// patchListToHere patches a chain of jumps to land at the current pc.
func (c *compiler) patchListToHere(list int) {
	if list == noJump {
		return
	}
	c.patchJump(list)
	// Follow jump chain through SJ field
	// For simplicity, we use a single jump per patch point (no chaining yet).
}

// concatJumpList links two jump lists. Returns the head.
func (fs *funcState) concatJumpList(l1, l2 int) int {
	if l2 == noJump {
		return l1
	}
	if l1 == noJump {
		return l2
	}
	// For our simple approach, we patch l1 to point to the same target as l2
	// This is simplified — real Lua uses linked lists through SJ fields.
	// For now, we handle this by storing both and patching both.
	// We'll track this externally.
	return l2 // simplified: only track latest
}

// ---------------------------------------------------------------------------
// Chunk compilation
// ---------------------------------------------------------------------------

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
	fs.proto.LastLine = lastLine

	c.leaveScope(lastLine)

	return c.closeFuncState()
}

// ---------------------------------------------------------------------------
// Block and statement compilation
// ---------------------------------------------------------------------------

func (c *compiler) compileBlock(block *ast.Block) {
	if block == nil {
		return
	}
	for _, stmt := range block.Stmts {
		c.compileStmt(stmt)
		// After each statement, release temporary registers while
		// preserving all registers occupied by active locals.
		// We use regTop() instead of nActVar because locals may not
		// occupy contiguous registers starting from R(0) — condition
		// temporaries (e.g., while/for loop conditions) can create gaps.
		c.fs.freeReg = c.fs.regTop()
	}
}

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
		c.compileLabelStmt(s)
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

func (c *compiler) compileLocalStmt(s *ast.LocalStmt) {
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
	fs.freeReg = base + nNames
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}
	fs.checkRegLimit()

	fs.checkVarLimit(nNames)
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

// isMultiRetExpr returns true if the expression can return multiple values
// (function calls and vararg ...)
func isMultiRetExpr(e ast.Expr) bool {
	switch e.(type) {
	case *ast.FuncCallExpr, *ast.MethodCallExpr, *ast.VarArgExpr:
		return true
	}
	return false
}

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

	// General case: evaluate all values into temp regs, then assign
	tempBase := fs.freeReg
	lastIsMultiRet := false

	for i := 0; i < nValues; i++ {
		if i == nValues-1 && nValues < nTargets {
			// Last expression, might be multi-return
			if isMultiRetExpr(s.Values[i]) {
				lastIsMultiRet = true
				c.compileExprMultiRet(s.Values[i], nTargets-i)
			} else {
				reg := tempBase + i
				c.compileExprToReg(s.Values[i], reg)
				if reg >= fs.freeReg {
					fs.freeReg = reg + 1
					if fs.freeReg > fs.maxReg {
						fs.maxReg = fs.freeReg
					}
				}
			}
		} else {
			reg := tempBase + i
			c.compileExprToReg(s.Values[i], reg)
			if reg >= fs.freeReg {
				fs.freeReg = reg + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			}
		}
	}

	// Fill missing values with nil (but not if last expr was multi-return)
	if nValues < nTargets && !lastIsMultiRet {
		for i := nValues; i < nTargets; i++ {
			reg := tempBase + i
			fs.emit(ABC(OP_LOADNIL, reg, 0, 0, 0), line)
			if reg >= fs.freeReg {
				fs.freeReg = reg + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			}
		}
	}

	// Now assign from temps to targets
	for i := 0; i < nTargets; i++ {
		c.assignToTarget(s.Targets[i], tempBase+i, line)
	}

	fs.freeReg = tempBase
}

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
			// For function calls, we need to use a temporary register because
			// compileFuncCall places the function into the target register before
			// evaluating arguments. If those arguments reference this variable,
			// they'd get the function instead of the original value.
			if _, isCall := value.(*ast.FuncCallExpr); isCall {
				tempReg := fs.reserveReg()
				c.compileExprToReg(value, tempReg)
				fs.emit(ABC(OP_MOVE, reg, tempReg, 0, 0), line)
				fs.freeReg = tempReg
				return
			}
			if _, isCall := value.(*ast.MethodCallExpr); isCall {
				tempReg := fs.reserveReg()
				c.compileExprToReg(value, tempReg)
				fs.emit(ABC(OP_MOVE, reg, tempReg, 0, 0), line)
				fs.freeReg = tempReg
				return
			}
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
		keyReg := fs.reserveReg()
		c.compileExprToReg(t.Key, keyReg)
		valReg := fs.reserveReg()
		c.compileExprToReg(value, valReg)
		fs.emit(ABC(OP_SETTABLE, tableReg, keyReg, valReg, 0), line)
		fs.freeReg = tableReg

	default:
		c.error(target, "invalid assignment target")
	}
}

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
		keyReg := fs.reserveReg()
		c.compileExprToReg(t.Key, keyReg)
		fs.emit(ABC(OP_SETTABLE, tableReg, keyReg, srcReg, 0), line)
		fs.freeReg = tableReg

	default:
		c.error(target, "invalid assignment target")
	}
}

func (c *compiler) compileSetGlobal(name string, value ast.Expr, line int) {
	fs := c.fs
	envUV := c.resolveEnv()
	nameK := fs.stringConstant(name)
	tempReg := fs.reserveReg()
	c.compileExprToReg(value, tempReg)
	fs.emitSetTabUp(envUV, nameK, tempReg, line)
	fs.freeReg = tempReg
}

func (c *compiler) resolveEnv() int {
	fs := c.fs
	if idx, ok := fs.lookupUpvalue("_ENV"); ok {
		return idx
	}
	idx, _ := c.resolveUpvalue(fs, "_ENV")
	return idx
}

// ---------------------------------------------------------------------------
// Expression statements (function calls)
// ---------------------------------------------------------------------------

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

func (c *compiler) compileReturnStmt(s *ast.ReturnStmt) {
	fs := c.fs
	line := s.P.Line

	if len(s.Values) == 0 {
		fs.emit(ABC(OP_RETURN0, 0, 0, 0, 0), line)
		return
	}

	if len(s.Values) == 1 {
		// Check for tail call
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
		}

		// Check for multi-return expression (vararg, method call)
		if isMultiRet(s.Values[0]) {
			base := fs.freeReg
			c.compileExprMultiRet(s.Values[0], 0) // 0 = all results
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
				c.compileExprMultiRet(val, 0) // 0 = all results
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

func (c *compiler) compileIfStmt(s *ast.IfStmt) {
	fs := c.fs
	line := s.P.Line

	var exitJumps []int

	// if cond then ...
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
}

// ---------------------------------------------------------------------------
// While
// ---------------------------------------------------------------------------

func (c *compiler) compileWhileStmt(s *ast.WhileStmt) {
	fs := c.fs
	line := s.P.Line

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
		fs.emit(ABC(OP_CLOSE, scope.nLocals, 0, 0, 0), line)
	}

	// Jump back to condition
	backJump := fs.emitJump(line)
	offset := loopStart - (fs.pc()) // negative
	fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)

	c.leaveScope(line)
	c.patchJump(exitJump)
}

// ---------------------------------------------------------------------------
// Repeat
// ---------------------------------------------------------------------------

func (c *compiler) compileRepeatStmt(s *ast.RepeatStmt) {
	fs := c.fs
	line := s.P.Line

	loopStart := fs.pc()
	fs.enterScope(true)

	c.compileBlock(s.Body)

	// Evaluate condition (may reference body locals)
	condLine := s.Cond.Pos().Line
	reg := fs.freeReg
	c.compileExprToReg(s.Cond, reg)

	// Close upvalues for body locals. OP_CLOSE captures values but does
	// not clear stack slots, so the condition result in reg remains valid.
	scope := fs.scopes[len(fs.scopes)-1]
	if fs.nActVar > scope.nLocals {
		fs.emit(ABC(OP_CLOSE, scope.nLocals, 0, 0, 0), condLine)
	}

	// If condition is falsy, jump back (keep looping)
	fs.emit(ABC(OP_TEST, reg, 0, 0, 0), condLine) // skip next if truthy → exit
	backJump := fs.emitJump(condLine)              // jump back (cond is false)
	offset := loopStart - fs.pc()
	fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)

	// Fall through here when condition is true (exit loop)
	c.leaveScope(line)
}

// ---------------------------------------------------------------------------
// Do block
// ---------------------------------------------------------------------------

func (c *compiler) compileDoStmt(s *ast.DoStmt) {
	c.fs.enterScope(false)
	c.compileBlock(s.Body)
	c.leaveScope(s.P.Line)
}

// ---------------------------------------------------------------------------
// Numeric for
// ---------------------------------------------------------------------------

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
	fs.checkVarLimit(4)
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

	// Close upvalues at the loop variable (base+3) and above before looping back.
	// This ensures each iteration gets its own closed upvalue copy of i.
	fs.emit(ABC(OP_CLOSE, base+3, 0, 0, 0), line)

	// FORLOOP — jumps back to just after FORPREP
	loopPC := fs.emit(ABx(OP_FORLOOP, base, 0), line)

	// Patch FORPREP to jump to FORLOOP
	bodyLen := loopPC - forPrepPC - 1
	fs.proto.Code[forPrepPC] = fs.proto.Code[forPrepPC].SetBx(bodyLen)

	// Patch FORLOOP to jump back
	fs.proto.Code[loopPC] = fs.proto.Code[loopPC].SetBx(bodyLen)

	c.leaveScope(line)
}

// ---------------------------------------------------------------------------
// Generic for
// ---------------------------------------------------------------------------

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

	// Mark base+3 as to-be-closed (the 4th return from iterator factory)
	fs.emit(ABC(OP_TBC, base+3, 0, 0, 0), line)

	// TFORPREP
	tforPrepPC := fs.emit(ABx(OP_TFORPREP, base, 0), line)

	// Add internal for-in variables as hidden locals to protect their registers
	fs.checkVarLimit(4 + len(s.Names))
	fs.locals = append(fs.locals,
		localVar{name: "(for state)", reg: base, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 1, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 2, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 3, startPC: fs.pc(), attrib: "close"},
	)
	fs.nActVar += 4

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
	fs.proto.Code[tforPrepPC] = fs.proto.Code[tforPrepPC].SetBx(bodyLen)

	// Patch TFORLOOP to jump back to loop body (after TFORPREP)
	backLen := tforLoopPC - tforPrepPC - 1
	fs.proto.Code[tforLoopPC] = fs.proto.Code[tforLoopPC].SetBx(backLen)

	c.leaveScope(line)
}

// ---------------------------------------------------------------------------
// Break / Goto / Labels
// ---------------------------------------------------------------------------

func (c *compiler) compileBreakStmt(s *ast.BreakStmt) {
	fs := c.fs
	scope := fs.findLoopScope()
	if scope == nil {
		c.error(s, "break outside loop")
		return
	}
	// Emit OP_CLOSE if there are close/captured locals being exited
	if fs.needsClose(scope.nLocals) {
		fs.emit(ABC(OP_CLOSE, scope.nLocals, 0, 0, 0), s.P.Line)
	}
	jpc := fs.emitJump(s.P.Line)
	scope.breakList = fs.concatJumpList(scope.breakList, jpc)
}

func (c *compiler) compileGotoStmt(s *ast.GotoStmt) {
	fs := c.fs
	line := s.P.Line

	// Check if label already exists (backward goto)
	for _, lbl := range fs.labels {
		if lbl.name == s.Label {
			// Emit OP_CLOSE if exiting scope with TBC/captured locals
			if fs.needsClose(lbl.nLocals) {
				fs.emit(ABC(OP_CLOSE, lbl.nLocals, 0, 0, 0), line)
			}
			jpc := fs.emitJump(line)
			offset := lbl.pc - (jpc + 1)
			fs.proto.Code[jpc] = fs.proto.Code[jpc].SetSJ(offset)
			return
		}
	}

	// Forward goto — emit placeholder OP_CLOSE and record it.
	// The OP_CLOSE operand will be patched when the label is resolved.
	closePC := -1
	if fs.needsClose(0) {
		// There are TBC/captured locals somewhere; emit placeholder
		closePC = fs.emit(ABC(OP_CLOSE, fs.nActVar, 0, 0, 0), line)
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

func (c *compiler) compileLabelStmt(s *ast.LabelStmt) {
	fs := c.fs

	// Check for duplicate label in current scope
	scope := fs.scopes[len(fs.scopes)-1]
	for _, lbl := range fs.labels[scope.firstLabel:] {
		if lbl.name == s.Name {
			c.error(s, "label '%s' already defined on line %d", s.Name, lbl.line)
			return
		}
	}

	fs.labels = append(fs.labels, labelInfo{
		name:    s.Name,
		pc:      fs.pc(),
		line:    s.P.Line,
		nLocals: fs.nActVar,
	})

	// Resolve pending gotos
	remaining := fs.pendGotos[:0]
	for _, pg := range fs.pendGotos {
		if pg.name == s.Name {
			// Validate: goto must not jump into scope of a local variable
			if pg.nLocals < fs.nActVar {
				c.error(s, "<goto %s> at line %d jumps into the scope of local variable", pg.name, pg.line)
				remaining = append(remaining, pg)
				continue
			}
			// Patch placeholder OP_CLOSE if one was emitted
			if pg.closePC >= 0 && fs.nActVar < pg.nLocals {
				fs.proto.Code[pg.closePC] = fs.proto.Code[pg.closePC].SetA(fs.nActVar)
			}
			offset := fs.pc() - (pg.pc + 1)
			fs.proto.Code[pg.pc] = fs.proto.Code[pg.pc].SetSJ(offset)
		} else {
			remaining = append(remaining, pg)
		}
	}
	fs.pendGotos = remaining
}

// ---------------------------------------------------------------------------
// Function statements
// ---------------------------------------------------------------------------

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
			fs.emit(ABC(OP_MOVE, localReg, reg, 0, 0), line)
		} else if uvIdx, ok := c.resolveUpvalue(fs, name.Name); ok {
			fs.emit(ABC(OP_SETUPVAL, reg, uvIdx, 0, 0), line)
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

func (c *compiler) compileLocalFuncStmt(s *ast.LocalFuncStmt) {
	fs := c.fs
	line := s.P.Line

	// Register the local first (allows recursion)
	fs.checkVarLimit(1)
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

func (c *compiler) compileGlobalStmt(s *ast.GlobalStmt) {
	// Treat like assignment to _ENV[name]
	fs := c.fs
	line := s.P.Line

	if s.Star {
		return // global * is a parser directive, no codegen
	}

	envUV := c.resolveEnv()
	for i, name := range s.Names {
		nameK := fs.stringConstant(name.Name)
		if i < len(s.Values) {
			reg := fs.reserveReg()
			c.compileExprToReg(s.Values[i], reg)
			fs.emitSetTabUp(envUV, nameK, reg, line)
			fs.freeReg = reg
		}
	}
}

func (c *compiler) compileGlobalFuncStmt(s *ast.GlobalFuncStmt) {
	fs := c.fs
	line := s.P.Line

	protoIdx := c.compileFunc(s.Func, line)
	reg := fs.reserveReg()
	fs.emit(ABx(OP_CLOSURE, reg, protoIdx), line)

	envUV := c.resolveEnv()
	nameK := fs.stringConstant(s.Name.Name)
	fs.emitSetTabUp(envUV, nameK, reg, line)

	fs.freeReg = reg
}

// ---------------------------------------------------------------------------
// Function compilation (body)
// ---------------------------------------------------------------------------

func (c *compiler) compileFunc(fe *ast.FuncExpr, line int) int {
	parentFS := c.fs

	source := parentFS.proto.Source
	fs := c.newFuncState(source, parentFS)
	fs.maxReg = 2

	fs.proto.LineDef = fe.P.Line
	fs.proto.NumParams = len(fe.Params)
	fs.proto.IsVarArg = fe.VarArg

	// _ENV upvalue (inherited from parent)
	parentEnv := -1
	if idx, ok := parentFS.lookupUpvalue("_ENV"); ok {
		parentEnv = idx
	}
	if parentEnv >= 0 {
		fs.addUpvalue("_ENV", false, parentEnv)
	}

	fs.enterScope(false)

	// Parameters are local variables
	fs.checkVarLimit(len(fe.Params))
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
	fs.checkRegLimit()

	// Vararg prep
	if fe.VarArg {
		fs.emit(ABC(OP_VARARGPREP, fs.proto.NumParams, 0, 0, 0), line)
	}

	c.compileBlock(fe.Body)

	// Ensure function ends with a return
	lastLine := line
	if fe.Body != nil && len(fe.Body.Stmts) > 0 {
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isMultiRet(e ast.Expr) bool {
	switch e.(type) {
	case *ast.FuncCallExpr, *ast.MethodCallExpr, *ast.VarArgExpr:
		return true
	}
	return false
}

func intLog2(x int) int {
	if x <= 0 {
		return 0
	}
	n := 0
	for x > 1 {
		x >>= 1
		n++
	}
	return n
}
