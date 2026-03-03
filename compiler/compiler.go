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
// FUTURE:
//
// tree-shake or refactor exprResult if we introduce an IR layer,
// basic-block graph?
//
// exprResult mirrors Lua 5.4’s expdesc model: a compact, single-pass
// lowering descriptor used during AST → bytecode emission.
//
// The structure intentionally overloads `info` and uses raw patch-list
// indices (`t`/`f`) for deferred jump resolution. This keeps emission
// fast and allocation-free, but relies on implicit invariants:
//
//   • The meaning of `info` depends strictly on `kind`.
//   • Only specific kinds may carry pending jump lists.
//   • Jump lists must be fully resolved before final register placement.
//   • Certain kinds (e.g. exprCall/exprVarArg/exprRelocate) require
//     post-processing before being treated as ordinary values.
//
// If the compiler evolves beyond single-pass emission (e.g. introduction
// of an IR layer, basic-block graph, SSA form, or optimization passes),
// this descriptor will likely need to be formalized or replaced with a
// more strongly-typed representation.
//
// Until then, changes to this type should preserve the current lowering
// invariants and be validated against short-circuit, call, and vararg
// edge cases.

// exprKind describes where an expression result resides after compilation.
type exprKind int

const (
	exprReg      exprKind = iota // value is in register info
	exprConst                    // value is constant index info
	exprTrue                     // literal true
	exprFalse                    // literal false
	exprNil                      // literal nil
	exprJump                     // conditional jump — info is index into pending jumps
	exprRelocate                 // instruction needs A set — info is pc
	exprCall                     // function call — info is pc of CALL instruction
	exprVarArg                   // vararg — info is pc of VARARG instruction
)

// exprResult tracks where an expression's value ended up after compilation.
type exprResult struct {
	kind exprKind
	info int // meaning depends on kind
	t    int // patch list for jumps-when-true  (-1 = no pending)
	f    int // patch list for jumps-when-false (-1 = no pending)
}

func (e exprResult) hasJumps() bool { return e.t != -1 || e.f != -1 }

// ---------------------------------------------------------------------------
// funcState — per-function compiler state
// ---------------------------------------------------------------------------

// localVar tracks a local variable during compilation.
type localVar struct {
	name     string
	reg      int
	startPC  int
	attrib   string // "", "const", "close"
	captured bool   // true if captured as an upvalue by an inner closure
}

// scopeInfo records the state at scope entry for restoration on scope exit.
type scopeInfo struct {
	nLocals    int   // number of locals when scope opened
	breakJumps []int // pending break jump PCs to be patched on scope exit
	isLoop     bool  // is this a loop scope?
	firstLabel int   // index into labels slice
	firstGoto  int   // index into pendGotos slice
}

// labelInfo records a ::label:: definition for goto resolution.
type labelInfo struct {
	name    string
	pc      int
	line    int
	nLocals int // number of active locals when label was defined
}

// pendingGoto records a forward goto that hasn't found its label yet.
type pendingGoto struct {
	name    string
	pc      int // jump instruction pc
	nLocals int // number of locals at goto
	line    int
	closePC int // pc of placeholder OP_CLOSE (-1 if none)
}

// funcState holds per-function compiler state: the proto being built,
// register allocator, local variables, scopes, and upvalue tracking.
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
	constLookup map[Value]int  // deduplication map for constants
}

// ---------------------------------------------------------------------------
// compiler — top-level state
// ---------------------------------------------------------------------------

// compiler is the top-level compilation state, holding the current funcState
// and accumulated error.
type compiler struct {
	fs     *funcState
	err    error
	limits CompilerLimits
}

// error records the first compilation error; subsequent errors are ignored.
func (c *compiler) error(pos interface{}, format string, args ...interface{}) {
	if c.err == nil {
		c.err = fmt.Errorf(format, args...)
	}
}

// ---------------------------------------------------------------------------
// Function state helpers
// ---------------------------------------------------------------------------

// newFuncState creates a new funcState for compiling a function body.
func (c *compiler) newFuncState(source string, parent *funcState) *funcState {
	fs := &funcState{
		c: c,
		proto: &Proto{
			Source: source,
		},
		parent:      parent,
		upvalLookup: make(map[string]int),
		constLookup: make(map[Value]int),
	}
	c.fs = fs
	return fs
}

// closeFuncState finalizes the current function's Proto, checks for
// unresolved gotos, and restores the parent funcState.
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

// reserveReg allocates the next free register and returns its index.
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

// reserveRegs allocates n consecutive registers and returns the base index.
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

// emit appends an instruction to the current proto and returns its pc.
func (fs *funcState) emit(inst Instruction, line int) int {
	pc := len(fs.proto.Code)
	fs.proto.Code = append(fs.proto.Code, inst)
	fs.proto.Lines = append(fs.proto.Lines, line)
	return pc
}

// pc returns the next instruction index (i.e. the current code length).
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

// addConstant adds a constant to the pool (deduplicating) and returns its index.
func (fs *funcState) addConstant(v Value) int {
	if idx, ok := fs.constLookup[v]; ok {
		return idx
	}
	idx := len(fs.proto.Constants)
	fs.proto.Constants = append(fs.proto.Constants, v)
	fs.constLookup[v] = idx
	return idx
}

// addProto adds a child prototype and returns its index (for OP_CLOSURE).
func (fs *funcState) addProto(p *Proto) int {
	idx := len(fs.proto.Protos)
	fs.proto.Protos = append(fs.proto.Protos, p)
	return idx
}

// stringConstant returns the constant index for a string, adding it if needed.
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

// addLocal declares a new local variable, reserves its register, and returns the register index.
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

// activateLocal marks a local variable as visible starting at the current pc.
func (fs *funcState) activateLocal(idx int) {
	if idx < len(fs.locals) {
		fs.locals[idx].startPC = fs.pc()
	}
}

// lookupLocal searches for an active local variable by name, returning its register.
func (fs *funcState) lookupLocal(name string) (int, bool) {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		if fs.locals[i].name == name && fs.locals[i].startPC >= 0 {
			return fs.locals[i].reg, true
		}
	}
	return 0, false
}

// needsClose returns true if any local at or above fromLocal is <close> or captured.
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

// isConst returns true if the named local has the <const> or <close> attribute.
// Both attributes make a variable immutable (Lua 5.4 §3.3.7).
func (fs *funcState) isConst(name string) bool {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		if fs.locals[i].name == name && fs.locals[i].startPC >= 0 {
			return fs.locals[i].attrib == "const" || fs.locals[i].attrib == "close"
		}
	}
	return false
}

// isConstUpvalue checks whether a captured variable is <const> in an enclosing scope.
func (c *compiler) isConstUpvalue(fs *funcState, name string) bool {
	if fs.parent == nil {
		return false
	}
	if fs.parent.isConst(name) {
		return true
	}
	return c.isConstUpvalue(fs.parent, name)
}

// markCaptured flags a local as captured by an inner closure's upvalue.
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

// lookupUpvalue checks if name is already registered as an upvalue in this function.
func (fs *funcState) lookupUpvalue(name string) (int, bool) {
	if idx, ok := fs.upvalLookup[name]; ok {
		return idx, true
	}
	return 0, false
}

// addUpvalue registers a new upvalue and returns its index.
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

// enterScope pushes a new scope. isLoop marks it as a loop (for break resolution).
func (fs *funcState) enterScope(isLoop bool) {
	fs.scopes = append(fs.scopes, scopeInfo{
		nLocals:    fs.nActVar,
		isLoop:     isLoop,
		firstLabel: len(fs.labels),
		firstGoto:  len(fs.pendGotos),
	})
}

// leaveScope pops the current scope, emits OP_CLOSE if needed, removes
// locals, resets registers, and patches pending break jumps.
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

	// Adjust pending gotos that originated inside this scope: lower their
	// nLocals to the scope's initial level so that the "jumps into scope"
	// check works correctly when the label is at a higher scope.
	// Also patch any OP_CLOSE placeholder to close down to the scope's
	// initial level. This matches Lua 5.4's movegotosout.
	for i := range fs.pendGotos {
		pg := &fs.pendGotos[i]
		if pg.nLocals > scope.nLocals {
			if pg.closePC >= 0 {
				fs.proto.Code[pg.closePC] = fs.proto.Code[pg.closePC].SetA(scope.nLocals)
			}
			pg.nLocals = scope.nLocals
		}
	}

	// Patch break jumps
	for _, jpc := range scope.breakJumps {
		c.patchJump(jpc)
	}
}

// findLoopScope walks the scope stack to find the innermost loop (for break).
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

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// resolveEnv returns the upvalue index for _ENV (the global environment table).
func (c *compiler) resolveEnv() int {
	fs := c.fs
	if idx, ok := fs.lookupUpvalue("_ENV"); ok {
		return idx
	}
	idx, _ := c.resolveUpvalue(fs, "_ENV")
	return idx
}

// isMultiRet returns true for expressions that can produce multiple results:
// function calls, method calls, and vararg (...).
func isMultiRet(e ast.Expr) bool {
	switch e.(type) {
	case *ast.FuncCallExpr, *ast.MethodCallExpr, *ast.VarArgExpr:
		return true
	}
	return false
}

// intLog2 returns floor(log2(x)), used for NEWTABLE hash size hints.
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
