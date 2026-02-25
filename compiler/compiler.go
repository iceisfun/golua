package compiler

import (
	"fmt"

	"github.com/iceisfun/golua/ast"
)

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Compile compiles a parsed block (chunk) into a top-level function prototype.
func Compile(source string, block *ast.Block) (*Proto, error) {
	c := &compiler{}
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
	name    string
	reg     int
	startPC int
	attrib  string // "", "const", "close"
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
}

type funcState struct {
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
	fs  *funcState
	err error
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
	return r
}

func (fs *funcState) reserveRegs(n int) int {
	base := fs.freeReg
	fs.freeReg += n
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
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

func (fs *funcState) addLocal(name string, attrib string) int {
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

	// Emit OP_CLOSE if this scope has any to-be-closed variables.
	// This closes upvalues and calls __close metamethods.
	if fs.nActVar > scope.nLocals {
		hasClose := false
		start := len(fs.locals) - (fs.nActVar - scope.nLocals)
		for i := start; i < len(fs.locals); i++ {
			if fs.locals[i].attrib == "close" {
				hasClose = true
				break
			}
		}
		if hasClose {
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
		fs.emit(ABC(OP_SETFIELD, tableReg, fieldK, valReg, 0), line)
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
			if reg != srcReg {
				fs.emit(ABC(OP_MOVE, reg, srcReg, 0, 0), line)
			}
			return
		}
		if idx, ok := c.resolveUpvalue(fs, t.Name); ok {
			fs.emit(ABC(OP_SETUPVAL, srcReg, idx, 0, 0), line)
			return
		}
		// Global: _ENV[name]
		envUV := c.resolveEnv()
		nameK := fs.stringConstant(t.Name)
		fs.emit(ABC(OP_SETTABUP, envUV, nameK, srcReg, 0), line)

	case *ast.FieldExpr:
		tableReg := fs.reserveReg()
		c.compileExprToReg(t.Table, tableReg)
		fieldK := fs.stringConstant(t.Field)
		fs.emit(ABC(OP_SETFIELD, tableReg, fieldK, srcReg, 0), line)
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
	fs.emit(ABC(OP_SETTABUP, envUV, nameK, tempReg, 0), line)
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

	// Test condition — exit when true, repeat when false
	condLine := s.Cond.Pos().Line
	reg := fs.freeReg
	c.compileExprToReg(s.Cond, reg)
	fs.emit(ABC(OP_TEST, reg, 0, 0, 1), condLine)    // skip next if truthy (continue loop)
	exitJump := fs.emitJump(condLine)                  // jump out (cond is true, stop repeating)
	backJump := fs.emitJump(condLine)                  // jump back (cond is false, repeat)
	offset := loopStart - fs.pc()
	fs.proto.Code[backJump] = fs.proto.Code[backJump].SetSJ(offset)

	c.patchJump(exitJump)
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

	// Add internal for loop variables as hidden locals to protect their registers
	// This ensures freeReg won't be reset below base+4 during the loop body
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

	// Compile iterator expressions into base, base+1, base+2
	nIter := len(s.Iters)
	if nIter == 1 && isMultiRet(s.Iters[0]) {
		// Single multi-return expression (e.g., pairs(t)) - ask for 3 results
		c.compileExprMultiRet(s.Iters[0], 3)
		fs.freeReg = base + 3
		if fs.freeReg > fs.maxReg {
			fs.maxReg = fs.freeReg
		}
	} else {
		for i, iter := range s.Iters {
			if i < 3 {
				c.compileExprToReg(iter, base+i)
				fs.freeReg = base + i + 1
				if fs.freeReg > fs.maxReg {
					fs.maxReg = fs.freeReg
				}
			}
		}
		// Fill missing with nil
		for i := nIter; i < 3; i++ {
			fs.emit(ABC(OP_LOADNIL, base+i, 0, 0, 0), line)
			fs.freeReg = base + i + 1
			if fs.freeReg > fs.maxReg {
				fs.maxReg = fs.freeReg
			}
		}
	}

	// Closing value (base+3) — nil
	fs.emit(ABC(OP_LOADNIL, base+3, 0, 0, 0), line)
	fs.freeReg = base + 4
	if fs.freeReg > fs.maxReg {
		fs.maxReg = fs.freeReg
	}

	// TFORPREP
	tforPrepPC := fs.emit(ABx(OP_TFORPREP, base, 0), line)

	// Add internal for-in variables as hidden locals to protect their registers
	fs.locals = append(fs.locals,
		localVar{name: "(for state)", reg: base, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 1, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 2, startPC: fs.pc()},
		localVar{name: "(for state)", reg: base + 3, startPC: fs.pc()},
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
	jpc := fs.emitJump(s.P.Line)
	scope.breakList = fs.concatJumpList(scope.breakList, jpc)
}

func (c *compiler) compileGotoStmt(s *ast.GotoStmt) {
	fs := c.fs
	line := s.P.Line

	// Check if label already exists (backward goto)
	for _, lbl := range fs.labels {
		if lbl.name == s.Label {
			// Validate: goto must not jump into scope of a local variable
			if fs.nActVar > lbl.nLocals {
				// This is a backward jump; the label had fewer locals,
				// but we need OP_CLOSE to handle upvalues for locals
				// being exited. This is valid — we're jumping OUT of scope.
			}
			jpc := fs.emitJump(line)
			offset := lbl.pc - (jpc + 1)
			fs.proto.Code[jpc] = fs.proto.Code[jpc].SetSJ(offset)
			return
		}
	}

	// Forward goto — record it
	jpc := fs.emitJump(line)
	fs.pendGotos = append(fs.pendGotos, pendingGoto{
		name:    s.Label,
		pc:      jpc,
		nLocals: fs.nActVar,
		line:    line,
	})
}

func (c *compiler) compileLabelStmt(s *ast.LabelStmt) {
	fs := c.fs

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
			fs.emit(ABC(OP_SETTABUP, envUV, nameK, reg, 0), line)
		}

	case *ast.FieldExpr:
		// Dotted name: a.b.c = function ...
		tableReg := fs.reserveReg()
		c.compileExprToReg(name.Table, tableReg)
		fieldK := fs.stringConstant(name.Field)
		fs.emit(ABC(OP_SETFIELD, tableReg, fieldK, reg, 0), line)
		fs.freeReg = tableReg
	}

	fs.freeReg = reg
}

func (c *compiler) compileLocalFuncStmt(s *ast.LocalFuncStmt) {
	fs := c.fs
	line := s.P.Line

	// Register the local first (allows recursion)
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
			fs.emit(ABC(OP_SETTABUP, envUV, nameK, reg, 0), line)
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
	fs.emit(ABC(OP_SETTABUP, envUV, nameK, reg, 0), line)

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
	case *ast.NilExpr:
		fs.emit(ABC(OP_LOADNIL, reg, 0, 0, 0), e.P.Line)

	case *ast.TrueExpr:
		fs.emit(ABC(OP_LOADTRUE, reg, 0, 0, 0), e.P.Line)

	case *ast.FalseExpr:
		fs.emit(ABC(OP_LOADFALSE, reg, 0, 0, 0), e.P.Line)

	case *ast.NumberExpr:
		if e.Value >= -OffsetSBx && e.Value <= OffsetSBx {
			fs.emit(AsBx(OP_LOADI, reg, int(e.Value)), e.P.Line)
		} else {
			k := fs.addConstant(IntValue(e.Value))
			fs.emit(ABx(OP_LOADK, reg, k), e.P.Line)
		}

	case *ast.FloatExpr:
		iv := int(e.Value)
		if float64(iv) == e.Value && iv >= -OffsetSBx && iv <= OffsetSBx {
			fs.emit(AsBx(OP_LOADF, reg, iv), e.P.Line)
		} else {
			k := fs.addConstant(FloatValue(e.Value))
			fs.emit(ABx(OP_LOADK, reg, k), e.P.Line)
		}

	case *ast.StringExpr:
		k := fs.addConstant(StringValue(e.Value))
		fs.emit(ABx(OP_LOADK, reg, k), e.P.Line)

	case *ast.NameExpr:
		c.compileName(e, reg)

	case *ast.BinopExpr:
		c.compileBinop(e, reg)

	case *ast.UnopExpr:
		c.compileUnop(e, reg)

	case *ast.FuncCallExpr:
		c.compileFuncCall(e, reg, 2, e.P.Line) // C=2 → 1 result

	case *ast.MethodCallExpr:
		c.compileMethodCall(e, reg, 2, e.P.Line)

	case *ast.FuncExpr:
		protoIdx := c.compileFunc(e, e.P.Line)
		fs.emit(ABx(OP_CLOSURE, reg, protoIdx), e.P.Line)

	case *ast.TableConstructor:
		c.compileTableConstructor(e, reg)

	case *ast.FieldExpr:
		c.compileFieldExpr(e, reg)

	case *ast.IndexExpr:
		c.compileIndexExpr(e, reg)

	case *ast.ParenExpr:
		c.compileExprToReg(e.Inner, reg)

	case *ast.VarArgExpr:
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
		fs.emit(ABC(OP_GETFIELD, reg, envReg, nameK, 0), e.P.Line)
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
	fs.emit(ABC(OP_GETTABUP, reg, envUV, nameK, 0), e.P.Line)
}

// ---------------------------------------------------------------------------
// Binary operations
// ---------------------------------------------------------------------------

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

	// Arithmetic / bitwise — compile both sides into registers
	leftReg := reg
	c.compileExprToReg(e.Left, leftReg)
	rightReg := fs.reserveReg()
	c.compileExprToReg(e.Right, rightReg)

	var op OpCode
	var mmOp int
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
	fs.emit(ABC(OP_MMBIN, leftReg, rightReg, mmOp, 0), line)
	fs.freeReg = rightReg
}

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

func (c *compiler) flattenConcat(e ast.Expr) []ast.Expr {
	if binop, ok := e.(*ast.BinopExpr); ok && binop.Op == ".." {
		left := c.flattenConcat(binop.Left)
		right := c.flattenConcat(binop.Right)
		return append(left, right...)
	}
	return []ast.Expr{e}
}

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

func (c *compiler) compileAnd(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	c.compileExprToReg(e.Left, reg)
	fs.emit(ABC(OP_TESTSET, reg, reg, 0, 0), line) // skip if falsy, keep value
	jmp := fs.emitJump(line)                        // jump to end (short-circuit)
	c.compileExprToReg(e.Right, reg)
	c.patchJump(jmp)
}

func (c *compiler) compileOr(e *ast.BinopExpr, reg int) {
	fs := c.fs
	line := e.P.Line

	c.compileExprToReg(e.Left, reg)
	fs.emit(ABC(OP_TESTSET, reg, reg, 0, 1), line) // skip if truthy, keep value
	jmp := fs.emitJump(line)                        // jump to end (short-circuit)
	c.compileExprToReg(e.Right, reg)
	c.patchJump(jmp)
}

// ---------------------------------------------------------------------------
// Unary operations
// ---------------------------------------------------------------------------

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
	fs.emit(ABC(OP_SELF, base, objReg, methodK, 0), line)

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
				fs.emit(ABC(OP_SETFIELD, reg, kIdx, valReg, 0), line)
				fs.freeReg = valReg
			case *ast.NameExpr:
				// name = value in table constructor
				valReg := fs.reserveReg()
				c.compileExprToReg(f.Value, valReg)
				kIdx := fs.stringConstant(key.Name)
				fs.emit(ABC(OP_SETFIELD, reg, kIdx, valReg, 0), line)
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

func (c *compiler) compileFieldExpr(e *ast.FieldExpr, reg int) {
	fs := c.fs
	tableReg := fs.reserveReg()
	c.compileExprToReg(e.Table, tableReg)
	fieldK := fs.stringConstant(e.Field)
	fs.emit(ABC(OP_GETFIELD, reg, tableReg, fieldK, 0), e.P.Line)
	fs.freeReg = tableReg
}

func (c *compiler) compileIndexExpr(e *ast.IndexExpr, reg int) {
	fs := c.fs
	tableReg := fs.reserveReg()
	c.compileExprToReg(e.Table, tableReg)
	keyReg := fs.reserveReg()
	c.compileExprToReg(e.Key, keyReg)
	fs.emit(ABC(OP_GETTABLE, reg, tableReg, keyReg, 0), e.P.Line)
	fs.freeReg = tableReg
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
