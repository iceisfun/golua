package compiler

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Lua 5.5 prototype flags (lobject.h), as carried by the dump format.
const (
	pfVaHid = 1 // PF_VAHID: function has hidden vararg arguments
	pfVaTab = 2 // PF_VATAB: function has a vararg table
	pfFixed = 4 // PF_FIXED: prototype has parts in fixed memory (load-time only)
)

// strippedSource is the source name given to a chunk dumped with strip=true.
// Reference leaves 'source' NULL and prints "?" for such functions; "=?" is
// the chunk-name spelling that produces the same "?" in GoLua's messages.
const strippedSource = "=?"

// Undump deserializes a Lua 5.5 binary chunk into a Proto. The layout is the
// one produced by reference ldump.c (and by luac5.5.0), so chunks precompiled
// by reference Lua load here unchanged.
// The data must start with the Lua binary signature ("\x1bLua").
// Returns the top-level Proto and the number of upvalues declared in the header.
func Undump(data []byte, source string) (proto *Proto, nUpvals int, retErr error) {
	u := &undumper{data: data, source: source}

	// Recover from panics (truncated chunks, overflow, etc.)
	defer func() {
		if r := recover(); r != nil {
			proto = nil
			nUpvals = 0
			if err, ok := r.(error); ok {
				retErr = err
			} else {
				retErr = fmt.Errorf("%v", r)
			}
		}
	}()

	if err := u.checkHeader(); err != nil {
		return nil, 0, err
	}
	nUp := int(u.readByte())
	// A stripped chunk carries no source name anywhere in the tree, so the
	// top-level fallback is the "?" spelling reference reports for it.
	p, err := u.loadFunction(strippedSource)
	if err != nil {
		return nil, 0, err
	}
	return p, nUp, nil
}

type undumper struct {
	data   []byte
	pos    int
	source string
	// saved mirrors the dumper's string-reuse table: each newly-read string is
	// appended (1-based index), and a size==0 marker is followed by the index of
	// a previously-read string to reuse. Must track the dumper's order exactly.
	saved []string
}

func (u *undumper) error(msg string) error {
	name := u.source
	if name == "" {
		name = "binary string"
	}
	return fmt.Errorf("%s: bad binary format (%s)", name, msg)
}

func (u *undumper) readByte() byte {
	if u.pos >= len(u.data) {
		panic(u.error("truncated chunk"))
	}
	b := u.data[u.pos]
	u.pos++
	return b
}

func (u *undumper) readBytes(n int) []byte {
	// n comes from chunk-controlled sizes, so guard against a negative or
	// wrapped length before it can turn the bounds check into a no-op.
	if n < 0 || n > len(u.data)-u.pos {
		panic(u.error("truncated chunk"))
	}
	b := u.data[u.pos : u.pos+n]
	u.pos += n
	return b
}

// readUnsigned reads a variable-length unsigned integer (Lua 5.5 format,
// lundump.c loadVarint). Each byte holds 7 data bits, most-significant group
// first; the high bit (0x80) marks a CONTINUATION byte, so the last byte of a
// value is the one with the high bit clear. Values above limit are rejected
// before the shift can wrap.
func (u *undumper) readUnsigned(limit uint64) uint64 {
	var x uint64
	lim := limit >> 7
	for {
		b := u.readByte()
		if x > lim {
			panic(u.error("integer overflow"))
		}
		x = (x << 7) | uint64(b&0x7f)
		if b&0x80 == 0 {
			break
		}
	}
	return x
}

// readSize reads a size/count. The limit is MaxInt64 rather than MaxUint64 so
// the result always fits a Go int on 64-bit platforms; anything larger is a
// malformed chunk regardless, since no chunk can be that long.
func (u *undumper) readSize() int {
	return int(u.readUnsigned(math.MaxInt64))
}

func (u *undumper) readInt() int {
	return int(u.readUnsigned(math.MaxInt32))
}

// readAlign skips the zero padding reference inserts so that a vector of
// native ints starts at a multiple of n bytes (lundump.c loadAlign).
func (u *undumper) readAlign(n int) {
	padding := n - u.pos%n
	if padding < n {
		u.readBytes(padding)
	}
}

// readCount reads an element count that will drive a slice allocation and
// verifies it cannot exceed the bytes remaining in the chunk. Every array
// element consumes at least one byte from the stream, so a count larger than
// the remaining input is necessarily a malformed chunk. Without this guard a
// corrupt count (up to ~2e9 via readInt) drives make([]T, count) straight into
// an uncatchable Go fatal OOM — a sandbox escape, since load() accepts binary
// chunks and recover() does not catch runtime.throw OOM.
func (u *undumper) readCount() int {
	n := u.readInt()
	if n > len(u.data)-u.pos {
		panic(u.error("truncated chunk"))
	}
	return n
}

// readRawInt64 reads a native 8-byte little-endian lua_Integer. Only the
// header's num-info block uses this layout; integer constants are varints.
func (u *undumper) readRawInt64() int64 {
	raw := u.readBytes(8)
	return int64(binary.LittleEndian.Uint64(raw))
}

// readInteger reads a signed integer constant in the zigzag-style coding of
// lundump.c loadInteger: 2x for x >= 0, -2x - 1 for x < 0.
func (u *undumper) readInteger() int64 {
	cx := u.readUnsigned(math.MaxUint64)
	if cx&1 != 0 {
		return int64(^(cx >> 1))
	}
	return int64(cx >> 1)
}

func (u *undumper) readNumber() float64 {
	raw := u.readBytes(8)
	return math.Float64frombits(binary.LittleEndian.Uint64(raw))
}

// readStringN reads a nullable string. present is false for the "no string"
// marker (size==0 followed by index 0), which reference stores as a NULL
// TString — used for the source of a stripped function.
func (u *undumper) readStringN() (s string, present bool) {
	size := u.readSize()
	if size == 0 {
		// Reuse marker: followed by a 1-based saved index (0 means NULL).
		idx := u.readSize()
		if idx == 0 {
			return "", false
		}
		if idx > len(u.saved) {
			panic(u.error("invalid string index"))
		}
		return u.saved[idx-1], true
	}
	size-- // stored as len+1
	// The stream holds the terminating '\0' too (dumpString writes size+1
	// bytes); it is not part of the string.
	raw := u.readBytes(size + 1)
	s = string(raw[:size])
	u.saved = append(u.saved, s)
	return s, true
}

// readString reads a string, treating an absent one as empty (GoLua has no
// nil string in Proto debug info).
func (u *undumper) readString() string {
	s, _ := u.readStringN()
	return s
}

func (u *undumper) readInstruction() Instruction {
	raw := u.readBytes(4)
	return Instruction(binary.LittleEndian.Uint32(raw))
}

func (u *undumper) checkHeader() error {
	// Signature: \x1bLua
	sig := u.readBytes(4)
	if string(sig) != "\x1bLua" {
		return u.error("not a binary chunk")
	}
	// Version
	if u.readByte() != 0x55 {
		return u.error("version mismatch")
	}
	// Format
	if u.readByte() != 0 {
		return u.error("format mismatch")
	}
	// LUAC_DATA
	luacData := u.readBytes(6)
	if string(luacData) != "\x19\x93\r\n\x1a\n" {
		return u.error("corrupted chunk")
	}
	// Lua 5.5 num-info entries (ldump.c dumpHeader): each is a sizeof byte
	// followed by a sample value of that type.
	// (int) LUAC_INT == -0x5678
	if u.readByte() != 4 {
		return u.error("int size mismatch")
	}
	if int32(binary.LittleEndian.Uint32(u.readBytes(4))) != -0x5678 {
		return u.error("integer format mismatch")
	}
	// (Instruction) LUAC_INST == 0x12345678
	if u.readByte() != 4 {
		return u.error("Instruction size mismatch")
	}
	if binary.LittleEndian.Uint32(u.readBytes(4)) != 0x12345678 {
		return u.error("instruction format mismatch")
	}
	// (lua_Integer) LUAC_INT == -0x5678
	if u.readByte() != 8 {
		return u.error("lua_Integer size mismatch")
	}
	if u.readRawInt64() != -0x5678 {
		return u.error("integer format mismatch")
	}
	// (lua_Number) LUAC_NUM == -370.5
	if u.readByte() != 8 {
		return u.error("lua_Number size mismatch")
	}
	if u.readNumber() != -370.5 {
		return u.error("float format mismatch")
	}
	return nil
}

// Operand conventions that differ between the reference Lua 5.5 encoding and
// GoLua's VM. Everything else about an instruction word — opcode numbering,
// field widths, field positions — is identical, so only these four cases are
// translated at the format boundary. Translating here (rather than in the VM)
// keeps the running interpreter on its own conventions while the binary format
// stays exactly reference's; CodeFromRef and CodeToRef are inverses over every
// encoding CodeFromRef accepts, so a chunk survives a dump/load round-trip
// unchanged.
//
//   - OP_MMBIN/OP_MMBINI/OP_MMBINK carry a metamethod tag in C. Reference's
//     TM_* enum lists the fast-access tags (INDEX..EQ) first, so its TM_ADD is
//     6 while GoLua's is 0.
//   - OP_FORLOOP/OP_TFORLOOP carry a backward jump in Bx. Reference jumps
//     'pc -= Bx' from the instruction after the loop instruction; GoLua's VM
//     jumps 'pc -= Bx + 1', so its Bx is one smaller for the same target.
//   - OP_SETLIST counts differently: reference's vC is how many elements were
//     already stored (so the batch fills vC+1 .. vC+n), while GoLua's is the
//     1-based index of the first element. With the k flag set, reference
//     splits that count between vC and the following EXTRAARG in units of
//     MaxArgVC+1, while GoLua keeps the whole index in the EXTRAARG.
//     Reference's lcode.c luaK_setlist uses the single-word form for every
//     'nelems <= MAXARG_vC', so its vC runs 0..MaxArgVC and the first index it
//     names runs 1..MaxArgVC+1 — one past what a 10-bit field holds as a
//     number. GoLua's index is 1-based, so a vC of 0 is a free encoding, and
//     it means exactly that top index. See maxSetListOffsetInWord.
//   - OP_SHLI and OP_SHRI mean the opposite things: GoLua's VM implements
//     opcode SHLI as reference's SHRI ("R[A] := R[B] >> sC") and opcode SHRI as
//     reference's SHLI ("R[A] := sC << R[B]"). Its own code generator emits
//     neither opcode, so the two only ever arrive from a reference chunk; the
//     operands mean the same thing in both, so swapping the opcode is enough.

// refTMAdd is the reference ordinal of TM_ADD (ltm.h: it follows INDEX,
// NEWINDEX, GC, MODE, LEN and EQ).
const refTMAdd = 6

// vABC builds an IvABC instruction (the NEWTABLE/SETLIST form, with a 6-bit vB
// and a 10-bit vC in place of B and C).
func vABC(op OpCode, a, vb, vc, k int) Instruction {
	return Instruction(uint32(op)<<PosOP |
		uint32(a)<<PosA |
		uint32(vb)<<PosVB |
		uint32(vc)<<PosVC |
		uint32(k)<<PosK)
}

// ErrSetListOffset reports an OP_SETLIST whose reference encoding names a first
// index that GoLua's encoding of the same instruction cannot hold. Nothing in
// reference's single-word form reaches that far today (see
// maxSetListOffsetInWord), so this is a guard against an encoding change rather
// than a case a stock toolchain produces. Loading such a chunk fails with an
// ordinary "bad binary format" error, which is catchable, rather than leaving
// the instruction in reference's encoding for GoLua's VM to misread — that
// silently stored the batch one slot low.
var ErrSetListOffset = fmt.Errorf(
	"OP_SETLIST first index exceeds this encoding's limit of %d", MaxArgVC+1)

// maxSetListOffsetInWord is the largest OP_SETLIST first index that fits in the
// instruction word itself. GoLua's vC holds that index directly and is 10 bits
// wide, but the index is 1-based, so a vC of 0 is free and carries the one
// value past the field's range; the VM (vm/vm_exec.go, case OP_SETLIST) decodes
// it that way. That is exactly the reach of reference's single-word form, whose
// vC counts the elements already stored rather than naming the next one:
// lcode.c luaK_setlist uses it for every 'nelems <= MAXARG_vC', so the first
// index it can name is MaxArgVC+1.
//
// A constructor gets there when its flush size divides MaxArgVC so the running
// count lands on it exactly. Reference's lparser.c maxtostore() drops to one
// element per SETLIST once the enclosing function is register-starved, and
// 'nelems' then steps through every value including MaxArgVC — luac emits
// "SETLIST A 1 1023" with no k flag for the 1024th element of a constructor
// written under ~180 locals.
const maxSetListOffsetInWord = MaxArgVC + 1

// CodeFromRef rewrites an instruction vector from the reference Lua 5.5
// encoding into GoLua's, in place. It returns an error for an instruction whose
// operands GoLua's encoding cannot represent; the caller turns that into an
// ordinary load failure, so such a chunk is rejected rather than executed with
// operands that mean something else.
func CodeFromRef(code []Instruction) error {
	for i, inst := range code {
		switch op := inst.OpCode(); op {
		case OP_MMBIN, OP_MMBINI, OP_MMBINK:
			if tag, ok := MetamethodTagFromLua54(inst.C()); ok {
				code[i] = ABC(op, inst.A(), inst.B(), int(tag), inst.K())
			}
		case OP_FORLOOP, OP_TFORLOOP:
			// A back jump of zero would target the loop instruction itself and
			// cannot come from a compiler; leave such a value alone rather
			// than wrapping it around the Bx field.
			if bx := inst.Bx(); bx > 0 {
				code[i] = ABx(op, inst.A(), bx-1)
			}
		case OP_SETLIST:
			// Reference counts what is already in the table; GoLua names the
			// first index to write. The translation never changes how many
			// words the instruction occupies: a k form stays a k form (GoLua's
			// EXTRAARG holds the whole index, so it always fits), and a
			// single-word form must stay single-word.
			stored := int64(inst.VC())
			if inst.K() != 0 {
				if i+1 >= len(code) {
					return fmt.Errorf("OP_SETLIST with k flag has no OP_EXTRAARG")
				}
				// Reference splits the count between vC and the EXTRAARG in
				// units of MaxArgVC+1. Combine them in int64: the product
				// overflows a 32-bit int.
				stored += int64(code[i+1].Ax()) * (MaxArgVC + 1)
				if stored+1 > MaxArgAx {
					// Reference can name indices GoLua's single EXTRAARG
					// cannot. Refuse rather than truncate the index.
					return fmt.Errorf(
						"OP_SETLIST first index %d exceeds the EXTRAARG limit of %d",
						stored+1, MaxArgAx)
				}
				code[i] = vABC(op, inst.A(), inst.VB(), 0, 1)
				code[i+1] = Ax(OP_EXTRAARG, int(stored)+1)
			} else {
				// Single word: GoLua's vC holds the first index itself, with 0
				// standing for maxSetListOffsetInWord.
				first := int(stored) + 1
				if first > maxSetListOffsetInWord {
					return ErrSetListOffset
				}
				if first == maxSetListOffsetInWord {
					first = 0
				}
				code[i] = vABC(op, inst.A(), inst.VB(), first, 0)
			}
		case OP_SHLI:
			code[i] = ABC(OP_SHRI, inst.A(), inst.B(), inst.C(), inst.K())
		case OP_SHRI:
			code[i] = ABC(OP_SHLI, inst.A(), inst.B(), inst.C(), inst.K())
		}
	}
	return nil
}

// CodeToRef returns a copy of an instruction vector in the reference Lua 5.5
// encoding. It is the inverse of CodeFromRef.
func CodeToRef(code []Instruction) []Instruction {
	out := make([]Instruction, len(code))
	copy(out, code)
	for i, inst := range out {
		switch op := inst.OpCode(); op {
		case OP_MMBIN, OP_MMBINI, OP_MMBINK:
			if tag := MetamethodTag(inst.C()); tag >= TM_ADD && tag <= TM_SHR {
				out[i] = ABC(op, inst.A(), inst.B(), int(tag)+refTMAdd, inst.K())
			}
		case OP_FORLOOP, OP_TFORLOOP:
			if bx := inst.Bx(); bx+1 <= MaxArgBx {
				out[i] = ABx(op, inst.A(), bx+1)
			}
		case OP_SETLIST:
			// The exact inverse of CodeFromRef: GoLua's first index becomes
			// reference's count of elements already stored, and the word count
			// is again preserved.
			first := inst.VC()
			if inst.K() != 0 {
				if i+1 >= len(out) {
					continue
				}
				first = out[i+1].Ax()
				if first < 1 {
					continue // not an index any compiler emits
				}
			} else if first == 0 {
				first = maxSetListOffsetInWord
			}
			stored := first - 1
			if inst.K() != 0 {
				out[i] = vABC(op, inst.A(), inst.VB(), stored%(MaxArgVC+1), 1)
				out[i+1] = Ax(OP_EXTRAARG, stored/(MaxArgVC+1))
			} else {
				// first is at most maxSetListOffsetInWord (CodeFromRef refuses
				// anything larger, and the code generator switches to the k
				// form at the same point), so stored always fits vC.
				out[i] = vABC(op, inst.A(), inst.VB(), stored, 0)
			}
		case OP_SHLI:
			out[i] = ABC(OP_SHRI, inst.A(), inst.B(), inst.C(), inst.K())
		case OP_SHRI:
			out[i] = ABC(OP_SHLI, inst.A(), inst.B(), inst.C(), inst.K())
		}
	}
	return out
}

func (u *undumper) loadFunction(parentSource string) (*Proto, error) {
	p := &Proto{}

	// Line info
	p.LineDef = u.readInt()
	p.LastLine = u.readInt()

	// Function header
	p.NumParams = int(u.readByte())
	// Prototype flags (lobject.h). Reference's lparser.c setvararg gives every
	// vararg function PF_VAHID — its extra arguments stay hidden below the
	// frame and are read with OP_GETVARG — and lcode.c needvatab promotes that
	// to PF_VATAB only when the function actually needs the arguments as a
	// table (luaK_finish then clears PF_VAHID, so the two are exclusive).
	// PF_VATAB is the form GoLua implements: a vararg table in the register
	// right after the fixed parameters, which is how its VM represents a named
	// vararg ("... name"), so that construct needs no private encoding.
	// A PF_VAHID function keeps no table and reads its named vararg with
	// OP_GETVARG, which the VM answers from the frame's own vararg values.
	// PF_FIXED is a property of a loaded prototype, never of the dump, so it is
	// masked off just as reference lundump.c does; any other bit is ignored.
	flag := u.readByte() &^ pfFixed
	p.HasNamedVarArg = flag&pfVaTab != 0
	p.IsVarArg = flag&(pfVaHid|pfVaTab) != 0
	p.MaxStack = int(u.readByte())
	// Either vararg flag means the frame owns one register beyond the fixed
	// parameters, at index NumParams: reference's parlist creates a local there
	// (the given name for "... name", "(vararg table)" for a plain "...") and
	// reserves a register for it, so maxstacksize always exceeds numparams for
	// a vararg prototype. GoLua's VM writes that slot on entry — the vararg
	// table for PF_VATAB, a nil for PF_VAHID so stale data cannot show through
	// debug.getlocal — with no bound of its own, so a numparams that breaks the
	// invariant would push the write past the registers the frame allocates.
	// Reject such a chunk here, where it is still an ordinary catchable load
	// error, rather than let a hostile chunk reach the VM and corrupt the stack.
	if p.IsVarArg && p.NumParams >= p.MaxStack {
		return nil, u.error("corrupted vararg register")
	}
	if p.HasNamedVarArg {
		// The vararg table is the local right after the fixed parameters.
		p.VarArgReg = p.NumParams
	}
	p.HasVarArgSlot = flag&pfVaHid != 0

	// Instructions
	nCode := u.readCount()
	u.readAlign(4)
	p.Code = make([]Instruction, nCode)
	for i := 0; i < nCode; i++ {
		p.Code[i] = u.readInstruction()
	}
	if err := CodeFromRef(p.Code); err != nil {
		return nil, u.error(err.Error())
	}

	// Constants
	nK := u.readCount()
	p.Constants = make([]Value, nK)
	for i := 0; i < nK; i++ {
		t := u.readByte()
		switch t {
		case 0x00: // LUA_VNIL
			p.Constants[i] = NilValue()
		case 0x01: // LUA_VFALSE
			p.Constants[i] = BoolValue(false)
		case 0x11: // LUA_VTRUE
			p.Constants[i] = BoolValue(true)
		case 0x03: // LUA_VNUMINT
			p.Constants[i] = IntValue(u.readInteger())
		case 0x13: // LUA_VNUMFLT
			p.Constants[i] = FloatValue(u.readNumber())
		case 0x04, 0x14: // LUA_VSHRSTR, LUA_VLNGSTR
			s, present := u.readStringN()
			if !present {
				return nil, u.error("bad format for constant string")
			}
			p.Constants[i] = StringValue(s)
		default:
			return nil, u.error(fmt.Sprintf("bad constant type %d", t))
		}
	}

	// Upvalues
	nUpvals := u.readCount()
	p.Upvalues = make([]UpvalDesc, nUpvals)
	for i := 0; i < nUpvals; i++ {
		p.Upvalues[i].InStack = u.readByte() != 0
		p.Upvalues[i].Index = int(u.readByte())
		// The kind byte (lparser.h VDKREG/RDKCONST/RDKVAVAR/RDKTOCLOSE) is
		// compile-time information that nothing at run time reads, but it is
		// part of the chunk, so it is kept verbatim and written back out
		// unchanged. Dropping it made a load/dump round-trip differ from the
		// input for any chunk that captures a "<const>", "<close>" or vararg
		// parameter as an upvalue.
		p.Upvalues[i].Kind = u.readByte()
	}

	// Nested protos
	nProtos := u.readCount()
	p.Protos = make([]*Proto, nProtos)
	for i := 0; i < nProtos; i++ {
		// Nested functions precede the source name in 5.5, so a child that has
		// no source of its own (stripped chunk) falls back to the same name
		// this function would.
		sub, err := u.loadFunction(parentSource)
		if err != nil {
			return nil, err
		}
		p.Protos[i] = sub
	}

	// Source name (absent in a stripped chunk)
	if src, present := u.readStringN(); present {
		p.Source = src
	} else {
		p.Source = parentSource
	}

	// Debug info
	// Line info (one per instruction): signed byte deltas; an ABSLINEINFO
	// (-0x80) entry takes its absolute line from the abslineinfo table.
	nLineInfo := u.readCount()
	var deltas []int8
	if nLineInfo > 0 {
		deltas = make([]int8, nLineInfo)
		for i := 0; i < nLineInfo; i++ {
			deltas[i] = int8(u.readByte())
		}
	}

	// Absolute line info: reference dumps the AbsLineInfo array as raw memory,
	// so the entries are 4-byte-aligned pairs of native ints.
	nAbsLineInfo := u.readCount()
	absPCs := make([]int, nAbsLineInfo)
	absLines := make([]int, nAbsLineInfo)
	if nAbsLineInfo > 0 {
		u.readAlign(4)
		for i := 0; i < nAbsLineInfo; i++ {
			absPCs[i] = int(int32(binary.LittleEndian.Uint32(u.readBytes(4))))
			absLines[i] = int(int32(binary.LittleEndian.Uint32(u.readBytes(4))))
		}
	}

	if nLineInfo > 0 {
		p.Lines = make([]int, nLineInfo)
		prev := p.LineDef
		absIdx := 0
		for i := 0; i < nLineInfo; i++ {
			if deltas[i] == -0x80 {
				for absIdx < nAbsLineInfo && absPCs[absIdx] < i {
					absIdx++
				}
				if absIdx >= nAbsLineInfo || absPCs[absIdx] != i {
					// Malformed chunk: marker with no matching absolute
					// entry. Keep this a catchable load error.
					panic(u.error("bad absolute line info"))
				}
				prev = absLines[absIdx]
				absIdx++
			} else {
				prev += int(deltas[i])
			}
			p.Lines[i] = prev
		}
	}

	// Local variables
	nLocVars := u.readCount()
	if nLocVars > 0 {
		p.Locals = make([]LocalVar, nLocVars)
		for i := 0; i < nLocVars; i++ {
			p.Locals[i].Name = u.readString()
			p.Locals[i].StartPC = u.readInt()
			p.Locals[i].EndPC = u.readInt()
		}
	}

	// Upvalue names
	nUpvalNames := u.readInt()
	if nUpvalNames != 0 {
		for i := 0; i < len(p.Upvalues); i++ {
			p.Upvalues[i].Name = u.readString()
		}
	}

	return p, nil
}
