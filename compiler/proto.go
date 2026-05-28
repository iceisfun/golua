package compiler

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// ---------------------------------------------------------------------------
// Constant values
// ---------------------------------------------------------------------------

// Value represents a Lua compile-time constant.
type Value struct {
	Type ValueType
	IVal int64
	FVal float64
	SVal string
}

// ValueType tags the kind of constant.
type ValueType byte

const (
	ValNil    ValueType = iota
	ValFalse            // boolean false
	ValTrue             // boolean true
	ValInt              // integer
	ValFloat            // float
	ValString           // string
)

// NilValue returns a nil constant.
func NilValue() Value { return Value{Type: ValNil} }

// BoolValue returns a boolean constant.
func BoolValue(b bool) Value {
	if b {
		return Value{Type: ValTrue}
	}
	return Value{Type: ValFalse}
}

// IntValue returns an integer constant.
func IntValue(v int64) Value { return Value{Type: ValInt, IVal: v} }

// FloatValue returns a floating-point constant.
func FloatValue(v float64) Value { return Value{Type: ValFloat, FVal: v} }

// StringValue returns a string constant.
func StringValue(s string) Value { return Value{Type: ValString, SVal: s} }

// String returns a human-readable representation of the constant value.
func (v Value) String() string {
	switch v.Type {
	case ValNil:
		return "nil"
	case ValFalse:
		return "false"
	case ValTrue:
		return "true"
	case ValInt:
		return fmt.Sprintf("%d", v.IVal)
	case ValFloat:
		if v.FVal == math.Trunc(v.FVal) && !math.IsInf(v.FVal, 0) {
			return fmt.Sprintf("%.1f", v.FVal)
		}
		return fmt.Sprintf("%g", v.FVal)
	case ValString:
		return fmt.Sprintf("%q", v.SVal)
	default:
		return "???"
	}
}

// ---------------------------------------------------------------------------
// Upvalue descriptor
// ---------------------------------------------------------------------------

// UpvalDesc describes how a closure captures an upvalue.
type UpvalDesc struct {
	Name    string
	InStack bool // true = captures from enclosing stack, false = from enclosing upvalue list
	Index   int  // register index (InStack) or upvalue index (!InStack)
}

// ---------------------------------------------------------------------------
// Debug info
// ---------------------------------------------------------------------------

// LocalVar records the name and scope of a local variable for debug info.
type LocalVar struct {
	Name    string
	StartPC int
	EndPC   int
}

// ---------------------------------------------------------------------------
// Proto — a compiled function prototype
// ---------------------------------------------------------------------------

// Proto is the compiled representation of a single Lua function (or the
// top-level chunk). It contains the instruction stream, constant pool,
// upvalue descriptors, and references to nested child prototypes.
//
// The VM executes a Proto by creating a [vm.Closure] that pairs the Proto
// with captured upvalues. The top-level chunk is itself a vararg function
// with a single upvalue (_ENV).
type Proto struct {
	Source   string // source file name (for error messages and debug info)
	LineDef  int    // first line of the function definition
	LastLine int    // last line of the function definition

	NumParams      int  // number of fixed (named) parameters
	IsVarArg       bool // true if the function accepts varargs (...)
	HasNamedVarArg bool // Lua 5.5: function has a named vararg parameter (... name)
	VarArgReg      int  // register index of the named vararg local

	MaxStack int // high-water mark of register usage (determines stack allocation)

	Code      []Instruction // instruction stream
	Lines     []int         // source line number for each instruction (parallel to Code)
	Constants []Value       // constant pool (strings, numbers, booleans, nil)
	Protos    []*Proto      // nested function prototypes (referenced by OP_CLOSURE)
	Upvalues  []UpvalDesc   // upvalue capture descriptors

	Locals []LocalVar // debug info: local variable names and scopes
}

// ---------------------------------------------------------------------------
// Disassembler — like luac -l
// ---------------------------------------------------------------------------

// Dump writes a luac-style disassembly to w.
func (p *Proto) Dump(w io.Writer) {
	p.dump(w, 0)
}

// DumpString returns the disassembly as a string.
func (p *Proto) DumpString() string {
	var buf strings.Builder
	p.Dump(&buf)
	return buf.String()
}

func (p *Proto) dump(w io.Writer, level int) {
	// Header
	va := ""
	if p.IsVarArg {
		va = "+"
	}
	fmt.Fprintf(w, "%sfunction <%s:%d,%d> (%d instructions)\n",
		strings.Repeat("  ", level),
		p.Source, p.LineDef, p.LastLine, len(p.Code))
	fmt.Fprintf(w, "%s%d params, %d slots, %d upvalues, %d locals, %d constants, %d functions%s\n",
		strings.Repeat("  ", level),
		p.NumParams, p.MaxStack, len(p.Upvalues),
		len(p.Locals), len(p.Constants), len(p.Protos), va)

	// Instructions
	for i, inst := range p.Code {
		line := 0
		if i < len(p.Lines) {
			line = p.Lines[i]
		}
		fmt.Fprintf(w, "%s\t%d\t[%d]\t%s\n",
			strings.Repeat("  ", level),
			i+1, line,
			p.formatInst(i, inst))
	}

	// Constants
	fmt.Fprintf(w, "%sconstants (%d):\n", strings.Repeat("  ", level), len(p.Constants))
	for i, k := range p.Constants {
		fmt.Fprintf(w, "%s\t%d\t%s\n", strings.Repeat("  ", level), i, k)
	}

	// Locals
	fmt.Fprintf(w, "%slocals (%d):\n", strings.Repeat("  ", level), len(p.Locals))
	for i, loc := range p.Locals {
		fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%d\n",
			strings.Repeat("  ", level), i, loc.Name, loc.StartPC+1, loc.EndPC+1)
	}

	// Upvalues
	fmt.Fprintf(w, "%supvalues (%d):\n", strings.Repeat("  ", level), len(p.Upvalues))
	for i, uv := range p.Upvalues {
		inStack := 0
		if uv.InStack {
			inStack = 1
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%d\n",
			strings.Repeat("  ", level), i, uv.Name, inStack, uv.Index)
	}

	// Sub-prototypes
	for _, sub := range p.Protos {
		fmt.Fprintln(w)
		sub.dump(w, level+1)
	}
}

func (p *Proto) formatInst(pc int, inst Instruction) string {
	op := inst.OpCode()
	name := OpName(op)

	switch GetOpMode(op) {
	case IABC:
		a, b, c, k := inst.A(), inst.B(), inst.C(), inst.K()
		extra := ""
		if k != 0 {
			extra = " ; k=1"
		}
		// Add annotations for specific opcodes
		switch op {
		case OP_LOADK:
			if inst.Bx() < len(p.Constants) {
				return fmt.Sprintf("%-12s %d %d\t; %s", name, a, inst.Bx(), p.Constants[inst.Bx()])
			}
		case OP_GETTABUP, OP_GETFIELD:
			if c < len(p.Constants) {
				return fmt.Sprintf("%-12s %d %d %d\t; %s", name, a, b, c, p.Constants[c].SVal)
			}
		case OP_SETTABUP, OP_SETFIELD:
			bName := ""
			if b < len(p.Constants) {
				bName = p.Constants[b].SVal
			}
			return fmt.Sprintf("%-12s %d %d %d%s\t; %s", name, a, b, c, extra, bName)
		case OP_SELF:
			if c < len(p.Constants) {
				return fmt.Sprintf("%-12s %d %d %d\t; %q", name, a, b, c, p.Constants[c].SVal)
			}
		case OP_LOADNIL:
			return fmt.Sprintf("%-12s %d %d", name, a, b)
		case OP_CALL, OP_TAILCALL:
			return fmt.Sprintf("%-12s %d %d %d", name, a, b, c)
		case OP_RETURN:
			return fmt.Sprintf("%-12s %d %d %d%s", name, a, b, c, extra)
		case OP_RETURN0:
			return fmt.Sprintf("%-12s", name)
		case OP_RETURN1:
			return fmt.Sprintf("%-12s %d", name, a)
		case OP_MOVE:
			return fmt.Sprintf("%-12s %d %d", name, a, b)
		case OP_LOADI:
			return fmt.Sprintf("%-12s %d %d", name, a, inst.SBx())
		case OP_LOADF:
			return fmt.Sprintf("%-12s %d %d", name, a, inst.SBx())
		case OP_TEST:
			return fmt.Sprintf("%-12s %d %d", name, a, k)
		case OP_TESTSET:
			return fmt.Sprintf("%-12s %d %d %d", name, a, b, k)
		case OP_EQ, OP_LT, OP_LE:
			return fmt.Sprintf("%-12s %d %d %d", name, a, b, k)
		case OP_EQI, OP_LTI, OP_LEI, OP_GTI, OP_GEI:
			return fmt.Sprintf("%-12s %d %d %d", name, a, inst.SB(), k)
		case OP_EQK:
			kstr := ""
			if b < len(p.Constants) {
				kstr = p.Constants[b].String()
			}
			return fmt.Sprintf("%-12s %d %d %d\t; %s", name, a, b, k, kstr)
		case OP_NOT, OP_UNM, OP_BNOT, OP_LEN:
			return fmt.Sprintf("%-12s %d %d", name, a, b)
		case OP_CONCAT:
			return fmt.Sprintf("%-12s %d %d", name, a, b)
		case OP_ADD, OP_SUB, OP_MUL, OP_MOD, OP_POW, OP_DIV, OP_IDIV,
			OP_BAND, OP_BOR, OP_BXOR, OP_SHL, OP_SHR:
			return fmt.Sprintf("%-12s %d %d %d", name, a, b, c)
		case OP_ADDK, OP_SUBK, OP_MULK, OP_MODK, OP_POWK, OP_DIVK, OP_IDIVK,
			OP_BANDK, OP_BORK, OP_BXORK:
			kstr := ""
			if c < len(p.Constants) {
				kstr = p.Constants[c].String()
			}
			return fmt.Sprintf("%-12s %d %d %d\t; %s", name, a, b, c, kstr)
		case OP_ADDI, OP_SHLI, OP_SHRI:
			return fmt.Sprintf("%-12s %d %d %d", name, a, b, inst.SC())
		case OP_MMBIN:
			return fmt.Sprintf("%-12s %d %d %d", name, a, b, c)
		case OP_MMBINI:
			return fmt.Sprintf("%-12s %d %d %d %d", name, a, inst.SB(), c, k)
		case OP_MMBINK:
			return fmt.Sprintf("%-12s %d %d %d %d", name, a, b, c, k)
		case OP_GETUPVAL, OP_SETUPVAL:
			return fmt.Sprintf("%-12s %d %d", name, a, b)
		case OP_GETTABLE, OP_SETTABLE:
			return fmt.Sprintf("%-12s %d %d %d%s", name, a, b, c, extra)
		case OP_GETI, OP_SETI:
			return fmt.Sprintf("%-12s %d %d %d%s", name, a, b, c, extra)
		case OP_VARARGPREP:
			return fmt.Sprintf("%-12s %d", name, a)
		case OP_VARARG:
			return fmt.Sprintf("%-12s %d %d", name, a, c)
		case OP_LOADFALSE, OP_LFALSESKIP, OP_LOADTRUE:
			return fmt.Sprintf("%-12s %d", name, a)
		case OP_CLOSE, OP_TBC:
			return fmt.Sprintf("%-12s %d", name, a)
		case OP_TFORCALL:
			return fmt.Sprintf("%-12s %d %d", name, a, c)
		}
		return fmt.Sprintf("%-12s %d %d %d %d", name, a, b, c, k)

	case IvABC:
		a := inst.A()
		// For NEWTABLE and SETLIST, use vB and vC extraction
		// vB is bits 16..21 (6 bits), vC is bits 22..31 (10 bits)
		vB := int((uint32(inst) >> 16) & 0x3F)
		vC := int((uint32(inst) >> 22) & 0x3FF)
		k := inst.K()
		switch op {
		case OP_NEWTABLE:
			return fmt.Sprintf("%-12s %d %d %d %d", name, a, vB, vC, k)
		case OP_SETLIST:
			return fmt.Sprintf("%-12s %d %d %d %d", name, a, vB, vC, k)
		}
		return fmt.Sprintf("%-12s %d %d %d %d", name, a, vB, vC, k)

	case IABx:
		a, bx := inst.A(), inst.Bx()
		switch op {
		case OP_LOADK:
			kstr := ""
			if bx < len(p.Constants) {
				kstr = p.Constants[bx].String()
			}
			return fmt.Sprintf("%-12s %d %d\t; %s", name, a, bx, kstr)
		case OP_CLOSURE:
			return fmt.Sprintf("%-12s %d %d", name, a, bx)
		case OP_FORLOOP, OP_FORPREP:
			return fmt.Sprintf("%-12s %d %d\t; to %d", name, a, bx, pc+2-bx+1)
		case OP_TFORPREP:
			return fmt.Sprintf("%-12s %d %d\t; to %d", name, a, bx, pc+2+bx)
		case OP_TFORLOOP:
			return fmt.Sprintf("%-12s %d %d\t; to %d", name, a, bx, pc+2-bx)
		}
		return fmt.Sprintf("%-12s %d %d", name, a, bx)

	case IAsBx:
		a, sbx := inst.A(), inst.SBx()
		switch op {
		case OP_LOADI:
			return fmt.Sprintf("%-12s %d %d", name, a, sbx)
		case OP_LOADF:
			return fmt.Sprintf("%-12s %d %d", name, a, sbx)
		}
		return fmt.Sprintf("%-12s %d %d", name, a, sbx)

	case IAx:
		ax := inst.Ax()
		return fmt.Sprintf("%-12s %d", name, ax)

	case IsJ:
		sj := inst.SJ()
		k := inst.K()
		target := pc + 1 + sj + 1
		_ = k
		return fmt.Sprintf("%-12s %d\t; to %d", name, sj, target)
	}

	return name
}
