package stdlib

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/vm"
)

// string.dump(function [, strip])
func stringDump(v *vm.VM) int {
	val := v.Get(1)
	if !val.IsFunction() && !val.IsNativeFunc() {
		callerArgError(v, 1, "string.dump", fmt.Sprintf("function expected, got %s", val.Type()))
	}
	if val.IsNativeFunc() {
		panic("unable to dump given function")
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

// dumpProto serializes a Proto to Lua 5.4 binary chunk format.
func dumpProto(p *compiler.Proto, strip bool) []byte {
	var buf bytes.Buffer
	d := &dumper{w: &buf, strip: strip}

	// Header
	buf.Write([]byte("\x1bLua")) // signature
	buf.WriteByte(0x54)          // version 5.4
	buf.WriteByte(0)             // format
	buf.Write([]byte("\x19\x93\r\n\x1a\n")) // LUAC_DATA
	buf.WriteByte(4)             // instruction size
	buf.WriteByte(8)             // integer size
	buf.WriteByte(8)             // number (float) size
	d.writeInt(0x5678)           // LUAC_INT check
	d.writeFloat(370.5)          // LUAC_NUM check

	// One upvalue for the top-level function
	buf.WriteByte(byte(len(p.Upvalues)))

	// Function
	d.dumpFunction(p)

	return buf.Bytes()
}

type dumper struct {
	w     *bytes.Buffer
	strip bool
}

func (d *dumper) writeByte(b byte) {
	d.w.WriteByte(b)
}

func (d *dumper) writeInt(n int64) {
	binary.Write(d.w, binary.LittleEndian, n)
}

func (d *dumper) writeFloat(f float64) {
	binary.Write(d.w, binary.LittleEndian, f)
}

func (d *dumper) writeUint32(n uint32) {
	binary.Write(d.w, binary.LittleEndian, n)
}

// writeSize writes a variable-length size using Lua 5.4's unsigned int encoding.
func (d *dumper) writeSize(n int) {
	d.writeVarInt(uint64(n))
}

func (d *dumper) writeVarInt(x uint64) {
	// Lua 5.4 uses a variable-length encoding for sizes:
	// Each byte holds 7 bits of data; the high bit is set on the last byte.
	if x == 0 {
		d.writeByte(0x80)
		return
	}
	var buf [10]byte
	i := 0
	for x > 0 {
		buf[i] = byte(x & 0x7f)
		x >>= 7
		i++
	}
	// Write in reverse order, setting high bit on last byte
	for j := i - 1; j >= 0; j-- {
		b := buf[j]
		if j == 0 {
			b |= 0x80 // mark last byte
		}
		d.writeByte(b)
	}
}

func (d *dumper) writeString(s string) {
	if s == "" {
		d.writeSize(0)
		return
	}
	d.writeSize(len(s) + 1)
	d.w.WriteString(s)
}

func (d *dumper) dumpFunction(p *compiler.Proto) {
	// Source name
	if d.strip {
		d.writeString("")
	} else {
		d.writeString(p.Source)
	}

	// Line info (variable-length encoded, matching Lua 5.4's dumpInt)
	d.writeSize(p.LineDef)
	d.writeSize(p.LastLine)

	// Function header
	d.writeByte(byte(p.NumParams))
	if p.IsVarArg {
		d.writeByte(1)
	} else {
		d.writeByte(0)
	}
	d.writeByte(byte(p.MaxStack))

	// Instructions
	d.writeSize(len(p.Code))
	for _, inst := range p.Code {
		d.writeUint32(uint32(inst))
	}

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
			d.writeByte(0x03) // LUA_VINTEGER (NUMINT)
			d.writeInt(k.IVal)
		case compiler.ValFloat:
			d.writeByte(0x13) // LUA_VNUMFLT
			d.writeFloat(k.FVal)
		case compiler.ValString:
			if len(k.SVal) < 40 {
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
		d.writeByte(0) // kind
	}

	// Nested protos
	d.writeSize(len(p.Protos))
	for _, sub := range p.Protos {
		d.dumpFunction(sub)
	}

	// Debug info
	if d.strip {
		d.writeSize(0) // lineinfo
		d.writeSize(0) // abslineinfo
		d.writeSize(0) // locvars
		d.writeSize(0) // upvalnames
	} else {
		// Line info (one per instruction)
		d.writeSize(len(p.Lines))
		if len(p.Lines) > 0 {
			prev := p.LineDef
			for _, line := range p.Lines {
				d.writeByte(byte(int8(line - prev)))
				prev = line
			}
		}
		// Absolute line info (empty for simplicity)
		d.writeSize(0)
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
}
