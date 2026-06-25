package compiler

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Undump deserializes a Lua 5.4 binary chunk into a Proto.
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
	p, err := u.loadFunction("")
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
	if u.pos+n > len(u.data) {
		panic(u.error("truncated chunk"))
	}
	b := u.data[u.pos : u.pos+n]
	u.pos += n
	return b
}

// readUnsigned reads a variable-length unsigned integer (Lua 5.4 format).
// Each byte holds 7 data bits; the high bit (0x80) is set on the LAST byte.
func (u *undumper) readUnsigned(limit uint64) uint64 {
	var x uint64
	lim := limit >> 7
	for {
		b := u.readByte()
		if x >= lim {
			panic(u.error("integer overflow"))
		}
		x = (x << 7) | uint64(b&0x7f)
		if b&0x80 != 0 {
			break
		}
	}
	return x
}

func (u *undumper) readSize() int {
	return int(u.readUnsigned(math.MaxUint64))
}

func (u *undumper) readInt() int {
	return int(u.readUnsigned(math.MaxInt32))
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

func (u *undumper) readInteger() int64 {
	raw := u.readBytes(8)
	return int64(binary.LittleEndian.Uint64(raw))
}

func (u *undumper) readNumber() float64 {
	raw := u.readBytes(8)
	return math.Float64frombits(binary.LittleEndian.Uint64(raw))
}

func (u *undumper) readStringN() string {
	size := u.readSize()
	if size == 0 {
		// Reuse marker: followed by a 1-based saved index (0 means NULL).
		idx := u.readSize()
		if idx == 0 {
			return ""
		}
		if idx > len(u.saved) {
			panic(u.error("invalid string index"))
		}
		return u.saved[idx-1]
	}
	size-- // stored as len+1
	s := string(u.readBytes(size))
	u.saved = append(u.saved, s)
	return s
}

func (u *undumper) readString() string {
	s := u.readStringN()
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
	if u.readInteger() != -0x5678 {
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

func (u *undumper) loadFunction(parentSource string) (*Proto, error) {
	p := &Proto{}

	// Source
	src := u.readStringN()
	if src == "" {
		p.Source = parentSource
	} else {
		p.Source = src
	}

	// Line info
	p.LineDef = u.readInt()
	p.LastLine = u.readInt()

	// Function header
	p.NumParams = int(u.readByte())
	// Vararg flag byte: bit 0 = has vararg, bit 1 = named vararg (... name),
	// bit 2 = reserves a (vararg table) slot at NumParams.
	vaFlag := u.readByte()
	p.IsVarArg = vaFlag&1 != 0
	p.HasNamedVarArg = vaFlag&2 != 0
	p.HasVarArgSlot = vaFlag&4 != 0
	if p.HasNamedVarArg {
		p.VarArgReg = int(u.readByte())
	}
	p.MaxStack = int(u.readByte())

	// Instructions
	nCode := u.readCount()
	p.Code = make([]Instruction, nCode)
	for i := 0; i < nCode; i++ {
		p.Code[i] = u.readInstruction()
	}
	// Lua 5.4 encodes MMBIN tags with its own TM_* ordinals (fast-access TMs
	// first, so TM_ADD=6). GoLua's own ordinals start at TM_ADD=0. Translate
	// the C field of MMBIN/MMBINI/MMBINK instructions from the reference
	// encoding to GoLua's, so later decoding (and traceback "metamethod 'X'"
	// labels) is unambiguous.
	for i, inst := range p.Code {
		op := inst.OpCode()
		if op != OP_MMBIN && op != OP_MMBINI && op != OP_MMBINK {
			continue
		}
		tag, ok := MetamethodTagFromLua54(inst.C())
		if !ok {
			continue
		}
		p.Code[i] = ABC(op, inst.A(), inst.B(), int(tag), inst.K())
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
			p.Constants[i] = StringValue(u.readString())
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
		u.readByte() // kind (unused in our Proto)
	}

	// Nested protos
	nProtos := u.readCount()
	p.Protos = make([]*Proto, nProtos)
	for i := 0; i < nProtos; i++ {
		sub, err := u.loadFunction(p.Source)
		if err != nil {
			return nil, err
		}
		p.Protos[i] = sub
	}

	// Debug info
	// Line info (one per instruction)
	nLineInfo := u.readCount()
	if nLineInfo > 0 {
		p.Lines = make([]int, nLineInfo)
		prev := p.LineDef
		for i := 0; i < nLineInfo; i++ {
			delta := int(int8(u.readByte()))
			prev += delta
			p.Lines[i] = prev
		}
	}

	// Absolute line info
	nAbsLineInfo := u.readInt()
	for i := 0; i < nAbsLineInfo; i++ {
		u.readInt() // pc
		u.readInt() // line
	}

	// Local variables
	nLocVars := u.readCount()
	if nLocVars > 0 {
		p.Locals = make([]LocalVar, nLocVars)
		for i := 0; i < nLocVars; i++ {
			p.Locals[i].Name = u.readStringN()
			p.Locals[i].StartPC = u.readInt()
			p.Locals[i].EndPC = u.readInt()
		}
	}

	// Upvalue names
	nUpvalNames := u.readInt()
	if nUpvalNames != 0 {
		for i := 0; i < len(p.Upvalues); i++ {
			p.Upvalues[i].Name = u.readStringN()
		}
	}

	return p, nil
}
