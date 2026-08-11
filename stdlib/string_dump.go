package stdlib

import (
	"bytes"
	"encoding/binary"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/vm"
)

// Lua 5.5 prototype flags (lobject.h). The dump format carries them verbatim
// in the byte that follows 'numparams'.
const (
	pfVaHid byte = 1 // PF_VAHID: function has hidden vararg arguments
	pfVaTab byte = 2 // PF_VATAB: function has a vararg table
	// PF_FIXED (4) describes a load-time property of the running prototype
	// (parts live in a fixed buffer), never a property of the dump, so it is
	// never written; reference lundump.c masks it off on load as well.
)

// string.dump(function [, strip])
func stringDump(v *vm.VM) int {
	val := v.Get(1)
	// Lua 5.5's str_dump uses a single luaL_argcheck: the argument must be a
	// Lua function (not a C/native function). A native function, wrong type,
	// or missing argument all produce the same "Lua function expected" error.
	if !val.IsFunction() {
		callerArgError(v, 1, "string.dump", "Lua function expected")
	}
	cl := val.AsClosure()
	strip := false
	if v.ArgCount() >= 2 && v.Get(2).ToBool() {
		strip = true
	}
	data := dumpProto(cl.Proto, strip)
	v.Set(0, vm.NewString(string(data)))
	return 1
}

// dumpProto serializes a Proto as a Lua 5.5 binary chunk. The layout is the
// one implemented by reference ldump.c, byte for byte: same header, same
// MSB-first varints, same 4-byte alignment before the code and absolute-line
// vectors, and the same field order inside a function (debug information and
// the source name last).
//
// The container format matches, so a chunk produced by luac5.5.0 survives a
// load-and-dump round trip byte for byte. The same is NOT true of a chunk
// GoLua compiled from source: the two code generators disagree about the
// instruction stream itself -- GoLua omits the EXTRAARG that must follow
// OP_NEWTABLE, does not apply luaK_finish's rewriting of a vararg function's
// final return, and orders constants differently, which also moves
// maxstacksize and the local-variable ranges. Reference Lua would therefore
// mis-execute a GoLua dump even though it parses cleanly, so do not feed one
// to it.
func dumpProto(p *compiler.Proto, strip bool) []byte {
	var buf bytes.Buffer
	d := &dumper{w: &buf, strip: strip, saved: map[string]int{}}

	// Header (ldump.c dumpHeader)
	buf.Write([]byte("\x1bLua"))            // signature
	buf.WriteByte(0x55)                     // version 5.5
	buf.WriteByte(0)                        // format
	buf.Write([]byte("\x19\x93\r\n\x1a\n")) // LUAC_DATA
	// Num-info entries: each is a sizeof byte followed by a sample value of
	// that type, written with native (little-endian) layout.
	buf.WriteByte(4)          // sizeof(int)
	d.writeUint32(0xFFFFA988) // (int) LUAC_INT = -0x5678
	buf.WriteByte(4)          // sizeof(Instruction)
	d.writeUint32(0x12345678) // (Instruction) LUAC_INST
	buf.WriteByte(8)          // sizeof(lua_Integer)
	d.writeRawInt64(-0x5678)  // (lua_Integer) LUAC_INT
	buf.WriteByte(8)          // sizeof(lua_Number)
	d.writeNumber(-370.5)     // (lua_Number) LUAC_NUM

	// Number of upvalues of the top-level function
	buf.WriteByte(byte(len(p.Upvalues)))

	// Function
	d.dumpFunction(p)

	return buf.Bytes()
}

type dumper struct {
	w     *bytes.Buffer
	strip bool
	// saved/nstr implement Lua 5.5 dump string reuse: each distinct string is
	// written once (assigned the next 1-based index in nstr) and every later
	// occurrence is emitted as size==0 followed by that saved index. This also
	// dedups the shared source name across the proto tree. Reference keys the
	// table by TString, which for both short and long strings compares by
	// content, so a Go map keyed by the string value reproduces its choices.
	saved map[string]int
	nstr  int
}

func (d *dumper) writeByte(b byte) {
	d.w.WriteByte(b)
}

// writeRawInt64 writes a native 8-byte little-endian lua_Integer (only used by
// the header's num-info block; constants use the varint encoding).
func (d *dumper) writeRawInt64(n int64) {
	binary.Write(d.w, binary.LittleEndian, n)
}

func (d *dumper) writeNumber(f float64) {
	binary.Write(d.w, binary.LittleEndian, f)
}

func (d *dumper) writeUint32(n uint32) {
	binary.Write(d.w, binary.LittleEndian, n)
}

// align pads with zero bytes until the offset from the start of the dump is a
// multiple of n, matching ldump.c dumpAlign. Reference reads the code and
// absolute-line vectors as native int arrays, so their start must be aligned.
func (d *dumper) align(n int) {
	padding := n - d.w.Len()%n
	if padding < n {
		for i := 0; i < padding; i++ {
			d.writeByte(0)
		}
	}
}

// writeVarint writes an unsigned integer using Lua 5.5's MSB-first varint
// encoding (ldump.c dumpVarint): 7 data bits per byte, most-significant group
// first, with the continuation bit 0x80 set on every byte except the last.
func (d *dumper) writeVarint(x uint64) {
	// 10 groups of 7 bits cover a 64-bit value (DIBS in ldump.c).
	var buf [10]byte
	n := 1
	buf[9] = byte(x & 0x7f) // least-significant byte, no continuation bit
	for x >>= 7; x != 0; x >>= 7 {
		n++
		buf[10-n] = byte(x&0x7f) | 0x80
	}
	d.w.Write(buf[10-n:])
}

// writeSize writes a non-negative count/size (ldump.c dumpSize/dumpInt).
func (d *dumper) writeSize(n int) {
	d.writeVarint(uint64(n))
}

// writeInteger writes a signed Lua integer constant using the zigzag-style
// coding of ldump.c dumpInteger, which keeps small negative values small:
// x >= 0 is coded as 2x, x < 0 as -2x - 1.
func (d *dumper) writeInteger(x int64) {
	var cx uint64
	if x >= 0 {
		cx = 2 * uint64(x)
	} else {
		cx = 2*^uint64(x) + 1
	}
	d.writeVarint(cx)
}

// writeNullString emits the "no string" marker (size==0, index==0), used for
// the source of a stripped function.
func (d *dumper) writeNullString() {
	d.writeVarint(0)
	d.writeVarint(0)
}

func (d *dumper) writeString(s string) {
	// Already saved? Emit size==0 plus the saved 1-based index (reuse).
	if idx, ok := d.saved[s]; ok {
		d.writeVarint(0)
		d.writeVarint(uint64(idx))
		return
	}
	// New string: size is len+1 (size==0 is reserved for the reuse marker),
	// followed by the bytes *including* the terminating '\0' that reference
	// stores in every TString. Save it under the next index for later reuse.
	d.writeSize(len(s) + 1)
	d.w.WriteString(s)
	d.writeByte(0)
	d.nstr++
	d.saved[s] = d.nstr
}

// dumpCode writes the instruction vector (ldump.c dumpCode): count, alignment
// padding, then the raw 4-byte words.
func (d *dumper) dumpCode(p *compiler.Proto) {
	d.writeSize(len(p.Code))
	d.align(4)
	// The instruction words go out in the reference operand encoding; the
	// undumper applies the inverse translation on the way back in.
	for _, inst := range compiler.CodeToRef(p.Code) {
		d.writeUint32(uint32(inst))
	}
}

func (d *dumper) dumpFunction(p *compiler.Proto) {
	d.writeSize(p.LineDef)
	d.writeSize(p.LastLine)
	d.writeByte(byte(p.NumParams))

	// Prototype flag byte. A named vararg ("... name") is not a GoLua
	// invention: Lua 5.5 has it too, and GoLua implements it exactly the way
	// reference's PF_VATAB describes — the extra arguments are collected into
	// a vararg table living in the register right after the fixed parameters,
	// and OP_VARARG reads from that table. So it needs no private flag bit and
	// no extra byte for the register: PF_VATAB says it, and numparams implies
	// the register. A plain "..." keeps its arguments hidden below the frame,
	// which is PF_VAHID.
	//
	// The two flags do not partition the same way in reference: lparser.c
	// setvararg marks every vararg function PF_VAHID, and lcode.c needvatab
	// upgrades it to PF_VATAB only when the vararg parameter is used as an
	// ordinary value. A reference function that only indexes its named vararg
	// stays PF_VAHID and reads it with OP_GETVARG. GoLua always materializes
	// the table, so it always writes PF_VATAB for a named vararg; that is a
	// faithful PF_VATAB prototype, but it is not the flag luac would have
	// chosen for every such function.
	var flag byte
	switch {
	case p.HasNamedVarArg:
		flag |= pfVaTab
	case p.IsVarArg:
		flag |= pfVaHid
	}
	d.writeByte(flag)

	d.writeByte(byte(p.MaxStack))

	d.dumpCode(p)

	// Constants
	d.writeSize(len(p.Constants))
	for _, k := range p.Constants {
		switch k.Type {
		case compiler.ValNil:
			d.writeByte(0x00) // LUA_VNIL
		case compiler.ValFalse:
			d.writeByte(0x01) // LUA_VFALSE
		case compiler.ValTrue:
			d.writeByte(0x11) // LUA_VTRUE
		case compiler.ValInt:
			d.writeByte(0x03) // LUA_VNUMINT
			d.writeInteger(k.IVal)
		case compiler.ValFloat:
			d.writeByte(0x13) // LUA_VNUMFLT
			d.writeNumber(k.FVal)
		case compiler.ValString:
			// The tag records the string's internal variant, which reference
			// picks by length: up to LUAI_MAXSHORTLEN (40) bytes is a short
			// (interned) string, above that a long one.
			if len(k.SVal) <= 40 {
				d.writeByte(0x04) // LUA_VSHRSTR
			} else {
				d.writeByte(0x14) // LUA_VLNGSTR
			}
			d.writeString(k.SVal)
		}
	}

	// Upvalues
	d.writeSize(len(p.Upvalues))
	for _, uv := range p.Upvalues {
		if uv.InStack {
			d.writeByte(1)
		} else {
			d.writeByte(0)
		}
		d.writeByte(byte(uv.Index))
		// The captured variable's kind (lparser.h): VDKREG, RDKCONST,
		// RDKVAVAR or RDKTOCLOSE. Nothing at run time reads it — reference's
		// own VM does not either — but it is part of the chunk, so a prototype
		// loaded from a reference chunk carries the byte it arrived with and
		// writes it back unchanged. GoLua's code generator does not classify
		// its upvalues yet, so a function it compiled dumps VDKREG (0), which
		// is what the field defaults to.
		d.writeByte(uv.Kind)
	}

	// Nested protos
	d.writeSize(len(p.Protos))
	for _, sub := range p.Protos {
		d.dumpFunction(sub)
	}

	// Source name. It comes after the nested protos in 5.5; the string-reuse
	// table dedups it across the whole tree. A stripped dump has no source.
	if d.strip {
		d.writeNullString()
	} else {
		d.writeString(p.Source)
	}

	d.dumpDebug(p)
}

func (d *dumper) dumpDebug(p *compiler.Proto) {
	if d.strip {
		d.writeSize(0) // lineinfo
		d.writeSize(0) // abslineinfo
		d.writeSize(0) // locvars
		d.writeSize(0) // upvalue names
		return
	}

	// Line info (one entry per instruction): signed byte deltas with an
	// ABSLINEINFO (-0x80) escape into the absolute table when a delta does not
	// fit in a signed byte or 128 instructions passed since the last anchor —
	// the conventions of lcode.c savelineinfo.
	type absLine struct{ pc, line int }
	var abs []absLine
	d.writeSize(len(p.Lines))
	if len(p.Lines) > 0 {
		prev := p.LineDef
		iwthabs := 0
		for pc, line := range p.Lines {
			delta := line - prev
			iwthabs++
			if delta <= -0x80 || delta >= 0x80 || iwthabs > 128 {
				abs = append(abs, absLine{pc: pc, line: line})
				iwthabs = 1
				delta = -0x80
			}
			d.writeByte(byte(int8(delta)))
			prev = line
		}
	}

	// Absolute line info: reference dumps the AbsLineInfo array as raw memory,
	// so the entries are aligned pairs of native ints.
	d.writeSize(len(abs))
	if len(abs) > 0 {
		d.align(4)
		for _, a := range abs {
			d.writeUint32(uint32(int32(a.pc)))
			d.writeUint32(uint32(int32(a.line)))
		}
	}

	// Local variables
	d.writeSize(len(p.Locals))
	for _, loc := range p.Locals {
		d.writeString(loc.Name)
		d.writeSize(loc.StartPC)
		d.writeSize(loc.EndPC)
	}

	// Upvalue names
	d.writeSize(len(p.Upvalues))
	for _, uv := range p.Upvalues {
		d.writeString(uv.Name)
	}
}
