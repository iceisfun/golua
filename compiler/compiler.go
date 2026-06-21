package compiler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iceisfun/golua/v2/ast"
)

// Local-variable attribute values. These mirror the strings carried in
// ast.LocalStmt.Attribs (produced by the parser from `<const>` / `<close>`).
const (
	attribNone  = ""      // a plain local with no attribute
	attribConst = "const" // `<const>`: read-only local
	attribClose = "close" // `<close>`: to-be-closed local
)

// envUpvalueName is the implicit upvalue holding the global environment (_ENV).
const envUpvalueName = "_ENV"

// forStateVarName is the debug name given to the hidden control registers of a
// for loop (the internal state/limit/step slots, not user-visible variables).
const forStateVarName = "(for state)"

// Repeated compiler diagnostics.
const (
	errControlStructureTooLong = "control structure too long"
	errAssignToConst           = "attempt to assign to const variable '%s'"
	errVarNotDeclared          = "variable '%s' not declared"
	errEnvIsGlobal             = "_ENV is global when accessing variable '%s'"
)

// shortSrc returns a display-friendly source name, matching Lua 5.4's luaO_chunkid.
func shortSrc(source string) string {
	if len(source) == 0 {
		return "[string \"?\"]"
	}
	switch source[0] {
	case '=':
		s := source[1:]
		if len(s) > 59 {
			s = s[:59]
		}
		return s
	case '@':
		s := source[1:]
		if len(s) >= 60 {
			s = "..." + s[len(s)-56:]
		}
		return s
	default:
		s := source
		truncated := false
		// Null byte truncates the name, matching Lua 5.4's C string behavior
		// where the null byte naturally terminates the C string. This is not
		// treated as "truncation" for the purposes of adding "..." suffix.
		if idx := strings.IndexByte(s, '\x00'); idx >= 0 {
			s = s[:idx]
		}
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
			truncated = true
		}
		if len(s) >= 45 {
			s = s[:45]
			truncated = true
		}
		if truncated {
			return "[string \"" + s + "...\"]"
		}
		return "[string \"" + s + "\"]"
	}
}

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
	endLine := block.EndLine
	if endLine == 0 {
		endLine = blockMaxLine(block)
	}
	c := &compiler{limits: cfg.limits.effective(), stringPool: make(map[string]string), endLine: endLine}
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

// String returns the exprKind's name for debugging and error output.
func (k exprKind) String() string {
	switch k {
	case exprReg:
		return "exprReg"
	case exprConst:
		return "exprConst"
	case exprTrue:
		return "exprTrue"
	case exprFalse:
		return "exprFalse"
	case exprNil:
		return "exprNil"
	case exprJump:
		return "exprJump"
	case exprRelocate:
		return "exprRelocate"
	case exprCall:
		return "exprCall"
	case exprVarArg:
		return "exprVarArg"
	default:
		return fmt.Sprintf("exprKind(%d)", int(k))
	}
}

// exprResult tracks where an expression's value ended up after compilation.
type exprResult struct {
	kind exprKind
	info int // meaning depends on kind
	t    int // patch list for jumps-when-true  (-1 = no pending)
	f    int // patch list for jumps-when-false (-1 = no pending)
}

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
	// inlined marks a `<const>` local whose initializer is a compile-time
	// foldable scalar (nil/bool/number/string). Inlined locals consume no
	// register (reg = -1), produce no debug-info entry, and are substituted
	// at use-sites with their constant value (matches Lua 5.5 semantics).
	inlined   bool
	inlineVal Value
}

// scopeInfo records the state at scope entry for restoration on scope exit.
type scopeInfo struct {
	nLocals        int   // number of locals when scope opened
	baseReg        int   // register base (freeReg) when scope opened — used for OP_CLOSE
	breakJumps     []int // pending break jump PCs to be patched on scope exit
	isLoop         bool  // is this a loop scope?
	firstLabel     int   // index into labels slice
	firstGoto      int   // index into pendGotos slice
	savedGlobalEnv globalEnv
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

	// globalBarrier records a Lua 5.5 global declaration (`global x` or
	// `global *`) that appeared after this goto in the same active block.
	// Reference Lua treats global declarations as scope-creating, so a goto
	// cannot jump past one into a non-block-end label. Empty when no barrier;
	// the special value "*" denotes a wildcard `global *` declaration.
	globalBarrier string
	// barrierDepth is the scope nesting depth (len(scopes)) at which the
	// barrier global was declared, used to clear it when that scope exits.
	barrierDepth int
}

// globalEnv tracks Lua 5.5 compile-time global declarations for the
// current scope. Every function starts with an implicit "global *"
// (explicit=false, star=true). Once an explicit global statement appears,
// undeclared names become errors.
type globalEnv struct {
	explicit bool              // any explicit global declaration seen in this scope chain
	names    map[string]string // declared name → attrib ("" or "const")
	star     bool              // wildcard * declared
	starAttr string            // "" (rw) or "const" (ro)

	// declOrder records, for each named `global X` declaration, the number of
	// entries in fs.locals at the moment the declaration was compiled. In
	// reference Lua 5.5 a `global X` declaration is inserted into the active
	// variable list (interleaved with locals) and name resolution scans that
	// list innermost-first, so a `global X` declared after a `local X` shadows
	// the local. golua keeps globals in a separate map; declOrder lets name
	// resolution reproduce that ordering: a local at fs.locals index i is
	// shadowed by a `global X` whose declOrder > i.
	declOrder map[string]int
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

	// closeLine is the line of the last token consumed for this funcState's
	// body — i.e. the 'end' keyword for inner functions, or the line of the
	// last statement for the main chunk. Used as the file:line prefix for
	// post-parse compile errors like unresolved gotos.
	closeLine int

	// blockLastLine is the AST end line of the most recently compiled
	// top-level statement in the current block. It tracks reference Lua's
	// ls->lastline at leaveblock time and is the line a block-exit OP_CLOSE
	// (which fires __close) is attributed to, matching reference diagnostics.
	blockLastLine int

	locals      []localVar
	scopes      []scopeInfo
	labels      []labelInfo
	pendGotos   []pendingGoto
	upvalues    []UpvalDesc
	upvalLookup map[string]int // name → index in upvalues
	constLookup map[Value]int  // deduplication map for constants
	globalEnv   globalEnv      // Lua 5.5 global declaration tracking
}

// ---------------------------------------------------------------------------
// compiler — top-level state
// ---------------------------------------------------------------------------

// compiler is the top-level compilation state, holding the current funcState
// and accumulated error.
type compiler struct {
	fs         *funcState
	err        error
	limits     CompilerLimits
	stringPool map[string]string // intern pool for string constants
	endLine    int               // last line of the source (for compile error messages)
}

// internString returns a string that shares the same backing memory as
// all other identical strings in this compilation unit. This ensures
// that string.format("%p", s) returns the same address for identical
// string constants across different functions.
func (c *compiler) internString(s string) string {
	if interned, ok := c.stringPool[s]; ok {
		return interned
	}
	c.stringPool[s] = s
	return s
}

// error records the first compilation error; subsequent errors are ignored.
func (c *compiler) error(pos interface{}, format string, args ...interface{}) {
	if c.err == nil {
		msg := fmt.Sprintf(format, args...)
		line := 0
		source := ""
		if c.fs != nil {
			source = c.fs.proto.Source
		}
		switch p := pos.(type) {
		case ast.Node:
			if p != nil {
				pp := p.Pos()
				line = pp.Line
				if source == "" && pp.Source != "" {
					source = pp.Source
				}
			}
		case int:
			line = p
		}
		if source != "" && line > 0 {
			c.err = fmt.Errorf("%s:%d: %s", shortSrc(source), line, msg)
		} else if source != "" {
			c.err = fmt.Errorf("%s: %s", shortSrc(source), msg)
		} else {
			c.err = fmt.Errorf("%s", msg)
		}
	}
}

// errorAtLine records a compilation error using the given line number for
// the file:line prefix.
func (c *compiler) errorAtLine(line int, format string, args ...interface{}) {
	if c.err == nil {
		msg := fmt.Sprintf(format, args...)
		source := ""
		if c.fs != nil {
			source = c.fs.proto.Source
		}
		if source != "" && line > 0 {
			c.err = fmt.Errorf("%s:%d: %s", shortSrc(source), line, msg)
		} else if source != "" {
			c.err = fmt.Errorf("%s: %s", shortSrc(source), msg)
		} else {
			c.err = fmt.Errorf("%s", msg)
		}
	}
}

// blockMaxLine returns the maximum line number found among the block's
// statements. This is used as a fallback end-line when WithEndLine is not
// provided, so that compiler errors include a source:line prefix.
func blockMaxLine(block *ast.Block) int {
	maxLine := 0
	if block != nil {
		for _, s := range block.Stmts {
			if l := s.Pos().Line; l > maxLine {
				maxLine = l
			}
		}
	}
	return maxLine
}

// blockLastStmtLine returns the source line of the final token of a block's
// last statement, which is the value reference Lua reports (ls->lastline) when
// it raises a "jumps into the scope" error at leaveblock. Returns 0 for an
// empty block.
func blockLastStmtLine(stmts []ast.Stmt) int {
	if len(stmts) == 0 {
		return 0
	}
	return stmtEndLine(stmts[len(stmts)-1])
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
	// Inherit the parent's global environment. In Lua 5.5, function bodies
	// do not start fresh — they carry the enclosing scope's global
	// declaration restrictions.
	if parent != nil {
		fs.globalEnv = parent.globalEnv
		if parent.globalEnv.names != nil {
			cp := make(map[string]string, len(parent.globalEnv.names))
			for k, v := range parent.globalEnv.names {
				cp[k] = v
			}
			fs.globalEnv.names = cp
		}
		// declOrder positions index the parent's locals list, which is
		// meaningless in the child. Inherited global declarations precede all
		// of the child's own locals, so map them to -1 ("declared before any
		// local here"). A local the child declares itself (index >= 0) then
		// correctly shadows the inherited global.
		if parent.globalEnv.declOrder != nil {
			cp := make(map[string]int, len(parent.globalEnv.declOrder))
			for k := range parent.globalEnv.declOrder {
				cp[k] = -1
			}
			fs.globalEnv.declOrder = cp
		}
	}
	c.fs = fs
	return fs
}

// closeFuncState finalizes the current function's Proto, checks for
// unresolved gotos, and restores the parent funcState.
func (c *compiler) closeFuncState() *Proto {
	fs := c.fs
	p := fs.proto

	// Check for unresolved gotos (label not visible). Lua reports the line
	// of the 'end' keyword for inner functions, or the line of the last
	// statement for the main chunk — captured in fs.closeLine.
	if len(fs.pendGotos) > 0 {
		errLine := fs.closeLine
		if errLine == 0 {
			errLine = c.endLine
		}
		for _, pg := range fs.pendGotos {
			c.errorAtLine(errLine, "no visible label '%s' for <goto> at line %d", pg.name, pg.line)
		}
	}

	// Close all remaining locals (skip inlined <const>: no debug entry)
	for i := range fs.locals {
		if fs.locals[i].startPC >= 0 && !fs.locals[i].inlined {
			p.Locals = append(p.Locals, LocalVar{
				Name:    fs.locals[i].name,
				StartPC: fs.locals[i].startPC,
				EndPC:   len(p.Code),
			})
		}
	}

	// Sort locals by StartPC to match Lua 5.4's expected ordering.
	// Scope-close appends locals in reverse register order; sorting
	// restores the register-order invariant that localName relies on.
	sort.SliceStable(p.Locals, func(i, j int) bool {
		return p.Locals[i].StartPC < p.Locals[j].StartPC
	})

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
//
// regBaseForLocals returns the first register past the first n active locals.
// This is the correct register operand for OP_CLOSE when closing down to
// n active locals, accounting for register gaps caused by temporaries.
func (fs *funcState) regBaseForLocals(n int) int {
	if n == 0 {
		return 0
	}
	start := len(fs.locals) - fs.nActVar
	end := start + n
	if end > len(fs.locals) {
		end = len(fs.locals)
	}
	top := 0
	for i := start; i < end; i++ {
		if r := fs.locals[i].reg + 1; r > top {
			top = r
		}
	}
	return top
}

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

// maxToStore computes the dynamic SETLIST flush threshold for table
// constructors based on the number of free registers. This follows the
// Lua 5.5 algorithm (lparser.c maxtostore) which replaces the fixed 50
// (LFIELDS_PER_FLUSH) to allow deeper constructor nesting.
func (fs *funcState) maxToStore() int {
	numFree := fs.c.limits.MaxRegs - fs.freeReg
	if numFree >= 160 {
		return numFree / 5
	}
	if numFree >= 80 {
		return 10
	}
	return 1
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

// fixLineAt changes the line number of the instruction at the given PC.
func (fs *funcState) fixLineAt(pc int, line int) {
	if pc >= 0 && pc < len(fs.proto.Lines) {
		fs.proto.Lines[pc] = line
	}
}

// isDischargeOp returns true if the opcode is a "discharge" instruction —
// one that loads a value into a register from a table, upvalue, or constant.
// In Lua 5.4's one-pass compiler, these instructions are emitted when an
// expression is "discharged" to a register, and they receive the current
// parser line (which may differ from the expression's source line).
func isDischargeOp(op OpCode) bool {
	switch op {
	case OP_GETI, OP_GETTABLE, OP_GETFIELD, OP_GETTABUP, OP_GETUPVAL,
		OP_LOADI, OP_LOADF, OP_LOADK, OP_LOADKX, OP_LOADNIL,
		OP_LOADTRUE, OP_LOADFALSE, OP_LFALSESKIP,
		OP_MOVE, OP_NEWTABLE:
		return true
	}
	return false
}

// lastEmittedLine returns the line number of the most recently emitted instruction.
func (c *compiler) lastEmittedLine() int {
	lines := c.fs.proto.Lines
	if len(lines) > 0 {
		return lines[len(lines)-1]
	}
	return 0
}

// fixDischargedLine adjusts the line of the last emitted instruction to
// match what Lua 5.4's one-pass compiler produces. In reference Lua,
// "discharge" instructions (GETI, GETUPVAL, LOADK, etc.) are emitted
// when an expression is materialized to a register, using the parser's
// current line — which for the left operand of a binary operator is the
// operator's line, not the operand's source line. This method replicates
// that behavior for golua's AST-based compiler.
func (c *compiler) fixDischargedLine(line int) {
	fs := c.fs
	pc := fs.pc() - 1
	if pc < 0 {
		return
	}
	op := OpCode(fs.proto.Code[pc] & 0x7F)
	if isDischargeOp(op) {
		fs.fixLineAt(pc, line)
	}
}

// loadConstant emits OP_LOADK or OP_LOADKX depending on the constant index size.
func (fs *funcState) loadConstant(reg int, kIdx int, line int) {
	if kIdx <= MaxArgBx {
		fs.emit(ABx(OP_LOADK, reg, kIdx), line)
	} else {
		fs.emit(ABx(OP_LOADKX, reg, 0), line)
		fs.emit(Ax(OP_EXTRAARG, kIdx), line)
	}
}

// emitGetTabUp emits OP_GETTABUP or, when kIdx > MaxArgC, a fallback
// sequence (GETUPVAL + LOADK/LOADKX + GETTABLE) that avoids 8-bit overflow.
func (fs *funcState) emitGetTabUp(reg, upIdx, kIdx int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_GETTABUP, reg, upIdx, kIdx, 0), line)
		return
	}
	saved := fs.freeReg
	tmpEnv := fs.reserveReg()
	tmpKey := fs.reserveReg()
	fs.emit(ABC(OP_GETUPVAL, tmpEnv, upIdx, 0, 0), line)
	fs.loadConstant(tmpKey, kIdx, line)
	fs.emit(ABC(OP_GETTABLE, reg, tmpEnv, tmpKey, 0), line)
	fs.freeReg = saved
}

// emitGetField emits OP_GETFIELD or, when kIdx > MaxArgC, a fallback
// sequence (LOADK/LOADKX + GETTABLE).
func (fs *funcState) emitGetField(reg, tableReg, kIdx int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_GETFIELD, reg, tableReg, kIdx, 0), line)
		return
	}
	saved := fs.freeReg
	tmpKey := fs.reserveReg()
	fs.loadConstant(tmpKey, kIdx, line)
	fs.emit(ABC(OP_GETTABLE, reg, tableReg, tmpKey, 0), line)
	fs.freeReg = saved
}

// emitSetTabUp emits OP_SETTABUP or, when kIdx > MaxArgC, a fallback
// sequence (GETUPVAL + LOADK/LOADKX + SETTABLE).
func (fs *funcState) emitSetTabUp(upIdx, kIdx, valReg int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_SETTABUP, upIdx, kIdx, valReg, 0), line)
		return
	}
	saved := fs.freeReg
	tmpEnv := fs.reserveReg()
	tmpKey := fs.reserveReg()
	fs.emit(ABC(OP_GETUPVAL, tmpEnv, upIdx, 0, 0), line)
	fs.loadConstant(tmpKey, kIdx, line)
	fs.emit(ABC(OP_SETTABLE, tmpEnv, tmpKey, valReg, 0), line)
	fs.freeReg = saved
}

// emitSetField emits OP_SETFIELD or, when kIdx > MaxArgC, a fallback
// sequence (LOADK/LOADKX + SETTABLE).
func (fs *funcState) emitSetField(tableReg, kIdx, valReg int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_SETFIELD, tableReg, kIdx, valReg, 0), line)
		return
	}
	saved := fs.freeReg
	tmpKey := fs.reserveReg()
	fs.loadConstant(tmpKey, kIdx, line)
	fs.emit(ABC(OP_SETTABLE, tableReg, tmpKey, valReg, 0), line)
	fs.freeReg = saved
}

// emitSelf emits OP_SELF or, when kIdx > MaxArgC, a fallback
// sequence (MOVE + LOADK/LOADKX + GETTABLE).
func (fs *funcState) emitSelf(base, objReg, kIdx int, line int) {
	if kIdx <= MaxArgC {
		fs.emit(ABC(OP_SELF, base, objReg, kIdx, 0), line)
		return
	}
	saved := fs.freeReg
	fs.emit(ABC(OP_MOVE, base+1, objReg, 0, 0), line)
	tmpKey := fs.reserveReg()
	fs.loadConstant(tmpKey, kIdx, line)
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
// The string is interned via the compiler's string pool so that identical
// literals across different functions share the same Go string backing.
func (fs *funcState) stringConstant(s string) int {
	return fs.addConstant(StringValue(fs.c.internString(s)))
}

// ---------------------------------------------------------------------------
// Local variables
// ---------------------------------------------------------------------------

// limitError raises a Lua 5.5 errorlimit-style message:
//
//	too many <what> (limit is <limit>) in <where>[ near '<token>']
//
// where <where> is "main function" or "function at line N", matching the
// reference compiler's errorlimit() + luaX_syntaxerror() near-clause.
func (fs *funcState) limitError(what string, limit, line int, near string) {
	msg := fmt.Sprintf("too many %s (limit is %d)", what, limit)
	if fs.proto.LineDef == 0 {
		msg += " in main function"
	} else {
		msg += fmt.Sprintf(" in function at line %d", fs.proto.LineDef)
	}
	if near != "" {
		if near == "<eof>" {
			msg += " near <eof>"
		} else {
			msg += fmt.Sprintf(" near '%s'", near)
		}
	}
	fs.c.error(line, "%s", msg)
}

// checkVarLimitAt checks that adding count new locals won't exceed the limit,
// with explicit source line and near-token context for Lua 5.4-style messages.
func (fs *funcState) checkVarLimitAt(count int, line int, near string) {
	if fs.nActVar+count > fs.c.limits.MaxVars {
		fs.limitError("local variables", fs.c.limits.MaxVars, line, near)
	}
}

// checkRegLimit checks that the current freeReg doesn't exceed the register limit.
func (fs *funcState) checkRegLimit() {
	if fs.freeReg > fs.c.limits.MaxRegs {
		fs.c.error(nil, "too many registers (limit is %d)", fs.c.limits.MaxRegs)
	}
}

// checkRegLimitAt is like checkRegLimit but emits the full Lua 5.5 errorlimit
// message (with "in function at line N near '<token>'") for the cases where the
// reference compiler reports the register limit during a declaration.
func (fs *funcState) checkRegLimitAt(line int, near string) {
	if fs.freeReg > fs.c.limits.MaxRegs {
		fs.limitError("registers", fs.c.limits.MaxRegs, line, near)
	}
}

// lookupLocal searches for an active local variable by name, returning its register.
// Inlined `<const>` locals have no register and are skipped here; callers that need
// to substitute their constant value should use lookupInlined first.
func (fs *funcState) lookupLocal(name string) (int, bool) {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		lv := fs.locals[i]
		if lv.name == name && lv.startPC >= 0 {
			if lv.inlined {
				return 0, false
			}
			// Lua 5.5: a `global name` declaration shadows an enclosing local
			// of the same name declared earlier. The global was recorded with
			// the locals-list length at its declaration; if that exceeds this
			// local's index, the global is "newer" and wins, so this local is
			// not visible (resolution falls through to _ENV[name]).
			if order, ok := fs.globalEnv.declOrder[name]; ok && order > i {
				return 0, false
			}
			return lv.reg, true
		}
	}
	return 0, false
}

// lookupInlined searches for an active inlined `<const>` local by name and
// returns its compile-time constant value. Returns ok=false if no such
// inlined local is in scope (the name might still be a regular local,
// upvalue, or global — caller continues normal resolution).
func (fs *funcState) lookupInlined(name string) (Value, bool) {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		lv := fs.locals[i]
		if lv.name == name && lv.startPC >= 0 {
			// A later `global name` declaration shadows this binding (Lua 5.5);
			// the name then resolves as a global, not the inlined constant.
			if order, ok := fs.globalEnv.declOrder[name]; ok && order > i {
				return Value{}, false
			}
			if lv.inlined {
				return lv.inlineVal, true
			}
			// A regular local with the same name shadows any inlined binding.
			return Value{}, false
		}
	}
	return Value{}, false
}

// needsClose returns true if any local at or above fromLocal is <close> or captured.
func (fs *funcState) needsClose(fromLocal int) bool {
	start := len(fs.locals) - (fs.nActVar - fromLocal)
	if start < 0 {
		start = 0
	}
	for i := start; i < len(fs.locals); i++ {
		if fs.locals[i].attrib == attribClose || fs.locals[i].captured {
			return true
		}
	}
	return false
}

// needsCloseTBC returns true if any local at or above fromLocal has the <close>
// attribute. Unlike needsClose, this does NOT check for captured upvalues.
// Used for tail call optimization: OP_TAILCALL already closes upvalues, so
// only <close> variables prevent tail calls.
func (fs *funcState) needsCloseTBC(fromLocal int) bool {
	start := len(fs.locals) - (fs.nActVar - fromLocal)
	if start < 0 {
		start = 0
	}
	for i := start; i < len(fs.locals); i++ {
		if fs.locals[i].attrib == attribClose {
			return true
		}
	}
	return false
}

// isConst returns true if the named local has the <const> or <close> attribute.
// Both attributes make a variable immutable (Lua 5.4 §3.3.7).
// Inlined locals are always const.
func (fs *funcState) isConst(name string) bool {
	for i := len(fs.locals) - 1; i >= 0; i-- {
		if fs.locals[i].name == name && fs.locals[i].startPC >= 0 {
			if fs.locals[i].inlined {
				return true
			}
			return fs.locals[i].attrib == attribConst || fs.locals[i].attrib == attribClose
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
		fs.c.error(nil, "too many upvalues (limit is %d) in function at line %d", fs.c.limits.MaxUpvals, fs.proto.LineDef)
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
	// Deep copy the current globalEnv so it can be restored on scope exit.
	saved := fs.globalEnv
	if saved.names != nil {
		cp := make(map[string]string, len(saved.names))
		for k, v := range saved.names {
			cp[k] = v
		}
		saved.names = cp
	}
	if saved.declOrder != nil {
		cp := make(map[string]int, len(saved.declOrder))
		for k, v := range saved.declOrder {
			cp[k] = v
		}
		saved.declOrder = cp
	}
	fs.scopes = append(fs.scopes, scopeInfo{
		nLocals:        fs.nActVar,
		baseReg:        fs.regTop(),
		isLoop:         isLoop,
		firstLabel:     len(fs.labels),
		firstGoto:      len(fs.pendGotos),
		savedGlobalEnv: saved,
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
			if fs.locals[i].attrib == attribClose || fs.locals[i].captured {
				needClose = true
				break
			}
		}
		if needClose {
			// For a plain block scope (do/if/else), the block-exit OP_CLOSE
			// that fires __close must be attributed to the block's last
			// statement, not the construct's closing token — reference Lua emits
			// this in leaveblock with ls->lastline. blockLastLine tracks the AST
			// end line of that statement. Loop scopes are excluded: their close
			// sites (per-iteration body close, and the generic-for iterator
			// to-be-closed slot) are emitted explicitly elsewhere and reference
			// attributes the loop-scope close to the loop header line, so they
			// keep the supplied line to preserve line-hook parity.
			closeLine := line
			if !scope.isLoop && fs.blockLastLine != 0 {
				closeLine = fs.blockLastLine
			}
			fs.emit(ABC(OP_CLOSE, scope.baseReg, 0, 0, 0), closeLine)
		}
	}

	// Remove locals from this scope.
	// Emit debug info in forward (register) order so localName works correctly,
	// then pop from the end as before.
	{
		nToRemove := fs.nActVar - scope.nLocals
		start := len(fs.locals) - nToRemove
		endPC := fs.pc()
		for i := start; i < len(fs.locals); i++ {
			if fs.locals[i].startPC >= 0 && !fs.locals[i].inlined {
				fs.proto.Locals = append(fs.proto.Locals, LocalVar{
					Name:    fs.locals[i].name,
					StartPC: fs.locals[i].startPC,
					EndPC:   endPC,
				})
			}
		}
		fs.locals = fs.locals[:start]
		fs.nActVar = scope.nLocals
	}

	// Reset freeReg past all remaining locals. We use regTop() instead
	// of scope.nLocals because the remaining locals may occupy registers
	// beyond their count (e.g., when a while loop condition temp creates
	// a gap between outer and inner locals).
	fs.freeReg = fs.regTop()

	// Restore globalEnv from before this scope.
	fs.globalEnv = scope.savedGlobalEnv

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
				fs.proto.Code[pg.closePC] = fs.proto.Code[pg.closePC].SetA(scope.baseReg)
			}
			pg.nLocals = scope.nLocals
		}
		// A global declared inside the closing scope goes out of scope, so a
		// goto that escapes that scope is no longer barred by it. The scope
		// has already been popped, so it sat at depth len(scopes)+1.
		if pg.globalBarrier != "" && pg.barrierDepth > len(fs.scopes) {
			pg.globalBarrier = ""
			pg.barrierDepth = 0
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
	if offset > MaxSJ || offset < MinSJ {
		c.error(nil, errControlStructureTooLong)
		return
	}
	fs.proto.Code[jpc] = fs.proto.Code[jpc].SetSJ(offset)
}

// markGlobalBarrier records a Lua 5.5 global declaration as a scope barrier
// for any goto pending at the current block level. Reference Lua treats a
// `global x` / `global *` declaration like a variable declaration: a goto that
// appears before it cannot jump past it into a (non-block-end) label, raising
// "<goto g> at line N jumps into the scope of 'x'" (or '*' for a wildcard).
// barrierName is the declared name, or "*" for a wildcard declaration.
func (c *compiler) markGlobalBarrier(barrierName string) {
	fs := c.fs
	depth := len(fs.scopes)
	for i := range fs.pendGotos {
		pg := &fs.pendGotos[i]
		// Only gotos at this block level (same nActVar) are affected; a goto
		// from a deeper, already-closed scope has been moved out and a goto
		// from an enclosing scope cannot target a label inside this one.
		if pg.globalBarrier == "" && pg.nLocals >= fs.nActVar {
			pg.globalBarrier = barrierName
			pg.barrierDepth = depth
		}
	}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// checkGlobalRead verifies that a global name is allowed to be read under
// the current Lua 5.5 global declaration rules.
func (c *compiler) checkGlobalRead(name string, node ast.Node) {
	ge := &c.fs.globalEnv
	if c.envDeclaredGlobal(name) {
		c.error(node, errEnvIsGlobal, name)
		return
	}
	if !ge.explicit {
		return
	}
	if _, ok := ge.names[name]; ok {
		return
	}
	if ge.star {
		return
	}
	c.error(node, errVarNotDeclared, name)
}

// checkGlobalWrite verifies that a global name is allowed to be written under
// the current Lua 5.5 global declaration rules.
func (c *compiler) checkGlobalWrite(name string, node ast.Node) {
	ge := &c.fs.globalEnv
	if c.envDeclaredGlobal(name) {
		c.error(node, errEnvIsGlobal, name)
		return
	}
	if !ge.explicit {
		return
	}
	if attrib, ok := ge.names[name]; ok {
		if attrib == attribConst {
			c.error(node, errAssignToConst, name)
		}
		return
	}
	if ge.star {
		if ge.starAttr == attribConst {
			c.error(node, errAssignToConst, name)
		}
		return
	}
	c.error(node, errVarNotDeclared, name)
}

// envDeclaredGlobal reports whether accessing global variable 'name' (a name
// other than _ENV itself) must fail because _ENV has been declared via a
// `global _ENV` statement. In Lua 5.5 a global access compiles to _ENV[name];
// if _ENV itself resolves as a global rather than an upvalue/local, the
// compiler raises "_ENV is global when accessing variable 'name'" (lparser.c
// buildglobal). A local/upvalue _ENV shadowing the declaration suppresses it.
func (c *compiler) envDeclaredGlobal(name string) bool {
	if name == envUpvalueName {
		return false
	}
	fs := c.fs
	if _, ok := fs.globalEnv.names[envUpvalueName]; !ok {
		return false
	}
	// A real local or upvalue _ENV shadows the global declaration.
	if _, ok := fs.lookupLocal(envUpvalueName); ok {
		return false
	}
	return true
}

// resolveEnv returns the upvalue index for _ENV (the global environment table).
func (c *compiler) resolveEnv() int {
	fs := c.fs
	if idx, ok := fs.lookupUpvalue(envUpvalueName); ok {
		return idx
	}
	idx, _ := c.resolveUpvalue(fs, envUpvalueName)
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
