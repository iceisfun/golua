package stdlib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/iceisfun/golua/v2/vm"
)

// getPackInt returns an integer argument for string.pack, coercing strings to
// numbers first (matching Lua 5.4 behavior). Reports "got nil" for missing args
// (not "got no value") and "number has no integer representation" for float strings.
func getPackInt(v *vm.VM, idx int) int64 {
	val := v.Get(idx)
	if i, ok := val.ToInt(); ok {
		return i
	}
	if val.IsNumber() {
		callerArgError(v, idx, "string.pack", "number has no integer representation")
	}
	// Coerce strings to numbers (Lua 5.4 does this for pack)
	if val.IsString() {
		if n, ok := val.ToNumber(); ok {
			// Got a number from string, try integer conversion
			i := int64(n)
			if float64(i) == n {
				return i
			}
			callerArgError(v, idx, "string.pack", "number has no integer representation")
		}
	}
	callerArgError(v, idx, "string.pack", fmt.Sprintf("number expected, got %s", v.ObjTypeName(val)))
	return 0 // unreachable
}

// getPackString returns a string argument for string.pack, coercing numbers.
// Unlike the generic getString, an absent argument is reported as "got nil"
// rather than "got no value": Lua 5.5's str_pack pushes a nil marker onto the
// stack after the format string, so a missing pack value reads back as nil.
// This mirrors getPackInt's handling of missing integer arguments.
func getPackString(v *vm.VM, idx int) string {
	val := v.Get(idx)
	if val.IsString() {
		return val.AsString()
	}
	if val.IsNumber() {
		return val.String()
	}
	callerArgError(v, idx, "string.pack", fmt.Sprintf("string expected, got %s", v.ObjTypeName(val)))
	return "" // unreachable
}

// packDirective describes a single parsed format directive.
type packDirective struct {
	kind     byte // 'i','I','f','d','n','c','z','s','x','X','<','>','=','!',' '
	size     int  // byte size for integers/floats/c/s, alignment for '!'
	signed   bool
	natAlign int // natural alignment of this directive
}

// parsePackSize reads an optional digit sequence after a format character.
// Returns the parsed size and updated index. If no digits, returns dflt.
func parsePackSize(fmt string, pos int, dflt int) (int, int) {
	if pos >= len(fmt) || fmt[pos] < '0' || fmt[pos] > '9' {
		return dflt, pos
	}
	n := 0
	for pos < len(fmt) && fmt[pos] >= '0' && fmt[pos] <= '9' {
		d := int(fmt[pos] - '0')
		if n > (maxPackSize-d)/10 {
			panic(luaFmtErr("invalid format"))
		}
		n = n*10 + d
		pos++
	}
	return n, pos
}

const maxPackSize = 0x7fffffff

func luaFmtErr(msg string) string {
	return fmt.Sprintf("bad argument #1 to 'string.pack' (string expected): %s", msg)
}

// getNatAlign returns the natural alignment for a data directive.
func getNatAlign(kind byte, size int) int {
	switch kind {
	case 'i', 'I':
		return size
	case 'h', 'H':
		return 2
	case 'l', 'L', 'j', 'J', 'T':
		return 8
	case 'f':
		return 4
	case 'd', 'n':
		return 8
	case 'b', 'B', 'c', 'z', 's', 'x':
		return 1
	}
	return 1
}

// getSize returns the size of a format directive (for non-variable-length ones).
func getSize(kind byte, size int) int {
	switch kind {
	case 'b', 'B':
		return 1
	case 'h', 'H':
		return 2
	case 'l', 'L', 'j', 'J', 'T':
		return 8
	case 'f':
		return 4
	case 'd', 'n':
		return 8
	case 'i', 'I':
		return size
	case 'c':
		return size
	case 'x':
		return 1
	case 's':
		return size // prefix size
	}
	return 0
}

// addPadding computes alignment padding bytes needed.
func addPadding(pos, align int) int {
	if align <= 1 {
		return 0
	}
	mod := pos % align
	if mod == 0 {
		return 0
	}
	return align - mod
}

// isPow2 checks if n is a power of 2.
func isPow2(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// walkFormat iterates over a Lua pack format string, calling fn for each data
// directive. Returns nothing; panics on format errors.
type formatState struct {
	byteOrder binary.ByteOrder
	maxAlign  int
	funcName  string // e.g. "string.pack" for error messages
}

func newFormatState(funcName string) *formatState {
	return &formatState{
		byteOrder: binary.LittleEndian, // native = little on x86/ARM
		maxAlign:  1,
		funcName:  funcName,
	}
}

// effectiveAlign computes the effective alignment for a directive with the
// given natural alignment. The effective alignment is min(maxAlign, natAlign).
// Lua 5.4 defers the power-of-2 check to this point rather than checking
// maxAlign at parse time, so !5 i4 works (effective=4) but !5 i8 fails (effective=5).
func (fs *formatState) effectiveAlign(natAlign int) int {
	align := natAlign
	if align > fs.maxAlign {
		align = fs.maxAlign
	}
	if !isPow2(align) {
		panic(fmt.Sprintf("bad argument #1 to '%s' (format asks for alignment not power of 2)", fs.funcName))
	}
	return align
}

// stringPack implements string.pack(fmt, v1, v2, ...)
func stringPack(v *vm.VM) int {
	format := getString(v, 1, "string.pack")
	argIdx := 2
	var buf bytes.Buffer
	fs := newFormatState("string.pack")

	i := 0
	for i < len(format) {
		ch := format[i]
		i++

		switch ch {
		case ' ':
			continue
		case '<':
			fs.byteOrder = binary.LittleEndian
			continue
		case '>':
			fs.byteOrder = binary.BigEndian
			continue
		case '=':
			fs.byteOrder = binary.LittleEndian // native
			continue
		case '!':
			var align int
			align, i = parsePackSize(format, i, 8) // default max alignment = 8
			if align < 1 || align > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", align))
			}
			fs.maxAlign = align
			continue
		}

		switch ch {
		case 'b', 'B':
			packInt(v, &buf, fs, ch, 1, ch == 'b', &argIdx)
		case 'h', 'H':
			packInt(v, &buf, fs, ch, 2, ch == 'h', &argIdx)
		case 'l', 'L':
			packInt(v, &buf, fs, ch, 8, ch == 'l', &argIdx)
		case 'j', 'J':
			packInt(v, &buf, fs, ch, 8, ch == 'j', &argIdx)
		case 'T':
			packInt(v, &buf, fs, ch, 8, false, &argIdx)
		case 'i', 'I':
			var size int
			size, i = parsePackSize(format, i, 4)
			if size < 1 || size > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
			}
			packInt(v, &buf, fs, ch, size, ch == 'i', &argIdx)
		case 'f':
			packFloat32(v, &buf, fs, &argIdx)
		case 'd', 'n':
			packFloat64(v, &buf, fs, ch, &argIdx)
		case 'c':
			var size int
			if i >= len(format) || format[i] < '0' || format[i] > '9' {
				panic("missing size for format option 'c'")
			}
			size, i = parsePackSize(format, i, 0)
			packFixedString(v, &buf, fs, size, &argIdx)
		case 'z':
			packZeroTermString(v, &buf, &argIdx)
		case 's':
			var size int
			size, i = parsePackSize(format, i, 8)
			if size < 1 || size > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
			}
			packSizedString(v, &buf, fs, size, &argIdx)
		case 'x':
			pad := addPadding(buf.Len(), fs.effectiveAlign(1))
			for p := 0; p < pad; p++ {
				buf.WriteByte(0)
			}
			buf.WriteByte(0)
		case 'X':
			natAlign := getXAlign(format, &i)
			align := fs.effectiveAlign(natAlign)
			pad := addPadding(buf.Len(), align)
			for p := 0; p < pad; p++ {
				buf.WriteByte(0)
			}
		default:
			panic(fmt.Sprintf("invalid format option '%c'", ch))
		}
	}

	v.Set(0, vm.NewString(buf.String()))
	return 1
}

// getXAlign parses the directive after X and returns its natural alignment.
func getXAlign(format string, i *int) int {
	if *i >= len(format) {
		panic("bad argument #1 to 'string.pack' (invalid next option for option 'X')")
	}
	ch := format[*i]
	*i++
	switch ch {
	case 'b', 'B':
		return 1
	case 'h', 'H':
		return 2
	case 'l', 'L', 'j', 'J', 'T':
		return 8
	case 'i', 'I':
		var size int
		size, *i = parsePackSize(format, *i, 4)
		if size < 1 || size > 16 {
			panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
		}
		return size
	case 'f':
		return 4
	case 'd', 'n':
		return 8
	case 'x':
		return 1
	case 's':
		var size int
		size, *i = parsePackSize(format, *i, 8)
		if size < 1 || size > 16 {
			panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
		}
		return size
	case 'c', 'z', '<', '>', '=', '!', 'X':
		panic("bad argument #1 to 'string.pack' (invalid next option for option 'X')")
	case ' ':
		// skip spaces and try next
		for *i < len(format) && format[*i] == ' ' {
			*i++
		}
		// but space is not a valid option after X
		panic("bad argument #1 to 'string.pack' (invalid next option for option 'X')")
	default:
		panic(fmt.Sprintf("invalid format option '%c'", ch))
	}
}

func packInt(v *vm.VM, buf *bytes.Buffer, fs *formatState, kind byte, size int, signed bool, argIdx *int) {
	natAlign := getNatAlign(kind, size)
	align := fs.effectiveAlign(natAlign)
	pad := addPadding(buf.Len(), align)
	for p := 0; p < pad; p++ {
		buf.WriteByte(0)
	}

	val := getPackInt(v, *argIdx)
	*argIdx++

	// Range check for sizes < 8
	if size < 8 {
		if signed {
			lo := -(int64(1) << (uint(size)*8 - 1))
			hi := (int64(1) << (uint(size)*8 - 1)) - 1
			if val < lo || val > hi {
				panic(fmt.Sprintf("bad argument #%d to 'string.pack' (integer overflow)", *argIdx-1))
			}
		} else {
			umax := uint64(1)<<(uint(size)*8) - 1
			if val < 0 || uint64(val) > umax {
				panic(fmt.Sprintf("bad argument #%d to 'string.pack' (unsigned overflow)", *argIdx-1))
			}
		}
	} else if size == 8 && !signed {
		// unsigned I8: value must be representable (any int64 bit pattern is valid as uint64)
		// but negative check not needed for I8 — Lua allows full 64-bit range via signed int64
	}

	// Encode
	var tmp [8]byte
	if fs.byteOrder == binary.LittleEndian {
		binary.LittleEndian.PutUint64(tmp[:], uint64(val))
	} else {
		binary.BigEndian.PutUint64(tmp[:], uint64(val))
	}

	if size <= 8 {
		if fs.byteOrder == binary.LittleEndian {
			buf.Write(tmp[:size])
		} else {
			buf.Write(tmp[8-size:])
		}
	} else {
		// size > 8: write the 8-byte two's-complement value, then extend.
		// Signed formats ('i') sign-extend a negative value with 0xFF;
		// unsigned formats ('I') always zero-extend. This mirrors Lua 5.5's
		// packint, which is called with neg=0 for the Kuint case — e.g.
		// string.pack("I16", -1) is ff*8 00*8, not ff*16.
		var ext byte
		if signed && val < 0 {
			ext = 0xFF
		}
		if fs.byteOrder == binary.LittleEndian {
			buf.Write(tmp[:8])
			for j := 8; j < size; j++ {
				buf.WriteByte(ext)
			}
		} else {
			for j := 8; j < size; j++ {
				buf.WriteByte(ext)
			}
			buf.Write(tmp[:8])
		}
	}
}

func packFloat32(v *vm.VM, buf *bytes.Buffer, fs *formatState, argIdx *int) {
	natAlign := 4
	align := fs.effectiveAlign(natAlign)
	pad := addPadding(buf.Len(), align)
	for p := 0; p < pad; p++ {
		buf.WriteByte(0)
	}

	val := getNumber(v, *argIdx, "string.pack")
	*argIdx++

	bits := math.Float32bits(float32(val))
	var tmp [4]byte
	if fs.byteOrder == binary.LittleEndian {
		binary.LittleEndian.PutUint32(tmp[:], bits)
	} else {
		binary.BigEndian.PutUint32(tmp[:], bits)
	}
	buf.Write(tmp[:])
}

func packFloat64(v *vm.VM, buf *bytes.Buffer, fs *formatState, kind byte, argIdx *int) {
	natAlign := 8
	align := fs.effectiveAlign(natAlign)
	pad := addPadding(buf.Len(), align)
	for p := 0; p < pad; p++ {
		buf.WriteByte(0)
	}

	val := getNumber(v, *argIdx, "string.pack")
	*argIdx++

	bits := math.Float64bits(val)
	var tmp [8]byte
	if fs.byteOrder == binary.LittleEndian {
		binary.LittleEndian.PutUint64(tmp[:], bits)
	} else {
		binary.BigEndian.PutUint64(tmp[:], bits)
	}
	buf.Write(tmp[:])
}

func packFixedString(v *vm.VM, buf *bytes.Buffer, fs *formatState, size int, argIdx *int) {
	s := getPackString(v, *argIdx)
	*argIdx++

	if len(s) > size {
		panic(fmt.Sprintf("bad argument #%d to 'string.pack' (string longer than given size)", *argIdx-1))
	}
	buf.WriteString(s)
	// pad with zeros
	for j := len(s); j < size; j++ {
		buf.WriteByte(0)
	}
}

func packZeroTermString(v *vm.VM, buf *bytes.Buffer, argIdx *int) {
	s := getPackString(v, *argIdx)
	*argIdx++

	if strings.ContainsRune(s, '\x00') {
		panic(fmt.Sprintf("bad argument #%d to 'string.pack' (string contains zeros)", *argIdx-1))
	}
	buf.WriteString(s)
	buf.WriteByte(0)
}

func packSizedString(v *vm.VM, buf *bytes.Buffer, fs *formatState, prefixSize int, argIdx *int) {
	s := getPackString(v, *argIdx)
	*argIdx++

	// Alignment for the prefix
	align := fs.effectiveAlign(prefixSize)
	pad := addPadding(buf.Len(), align)
	for p := 0; p < pad; p++ {
		buf.WriteByte(0)
	}

	slen := uint64(len(s))
	// Check if length fits in prefix
	if prefixSize < 8 {
		maxLen := uint64(1)<<(uint(prefixSize)*8) - 1
		if slen > maxLen {
			panic(fmt.Sprintf("bad argument #%d to 'string.pack' (string length does not fit in given size)", *argIdx-1))
		}
	}

	// Encode length as unsigned integer
	var tmp [8]byte
	if fs.byteOrder == binary.LittleEndian {
		binary.LittleEndian.PutUint64(tmp[:], slen)
	} else {
		binary.BigEndian.PutUint64(tmp[:], slen)
	}

	if prefixSize <= 8 {
		if fs.byteOrder == binary.LittleEndian {
			buf.Write(tmp[:prefixSize])
		} else {
			buf.Write(tmp[8-prefixSize:])
		}
	} else {
		if fs.byteOrder == binary.LittleEndian {
			buf.Write(tmp[:8])
			for j := 8; j < prefixSize; j++ {
				buf.WriteByte(0)
			}
		} else {
			for j := 8; j < prefixSize; j++ {
				buf.WriteByte(0)
			}
			buf.Write(tmp[:8])
		}
	}

	buf.WriteString(s)
}

// stringUnpack implements string.unpack(fmt, s [, pos])
func stringUnpack(v *vm.VM) int {
	format := getString(v, 1, "string.unpack")
	data := getString(v, 2, "string.unpack")
	pos := int64(1)
	if v.ArgCount() >= 3 && !v.Get(3).IsNil() {
		pos = getInt(v, 3, "string.unpack")
	}

	// Resolve negative position
	if pos < 0 {
		pos = int64(len(data)) + pos + 1
	}
	if pos < 1 {
		pos = 1
	}

	offset := int(pos) - 1 // 0-based byte offset
	if offset > len(data) {
		panic("bad argument #3 to 'string.unpack' (initial position out of string)")
	}

	fs := newFormatState("string.unpack")
	nret := 0

	i := 0
	for i < len(format) {
		ch := format[i]
		i++

		switch ch {
		case ' ':
			continue
		case '<':
			fs.byteOrder = binary.LittleEndian
			continue
		case '>':
			fs.byteOrder = binary.BigEndian
			continue
		case '=':
			fs.byteOrder = binary.LittleEndian
			continue
		case '!':
			var align int
			align, i = parsePackSize(format, i, 8)
			if align < 1 || align > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", align))
			}
			fs.maxAlign = align
			continue
		}

		switch ch {
		case 'b', 'B':
			val := unpackInt(data, &offset, fs, ch, 1, ch == 'b')
			v.Set(nret, val)
			nret++
		case 'h', 'H':
			val := unpackInt(data, &offset, fs, ch, 2, ch == 'h')
			v.Set(nret, val)
			nret++
		case 'l', 'L':
			val := unpackInt(data, &offset, fs, ch, 8, ch == 'l')
			v.Set(nret, val)
			nret++
		case 'j', 'J':
			val := unpackInt(data, &offset, fs, ch, 8, ch == 'j')
			v.Set(nret, val)
			nret++
		case 'T':
			val := unpackInt(data, &offset, fs, ch, 8, false)
			v.Set(nret, val)
			nret++
		case 'i', 'I':
			var size int
			size, i = parsePackSize(format, i, 4)
			if size < 1 || size > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
			}
			val := unpackInt(data, &offset, fs, ch, size, ch == 'i')
			v.Set(nret, val)
			nret++
		case 'f':
			val := unpackFloat32(data, &offset, fs)
			v.Set(nret, val)
			nret++
		case 'd', 'n':
			val := unpackFloat64(data, &offset, fs, ch)
			v.Set(nret, val)
			nret++
		case 'c':
			if i >= len(format) || format[i] < '0' || format[i] > '9' {
				panic("missing size for format option 'c'")
			}
			var size int
			size, i = parsePackSize(format, i, 0)
			val := unpackFixedString(data, &offset, size)
			v.Set(nret, val)
			nret++
		case 'z':
			val := unpackZeroTermString(data, &offset)
			v.Set(nret, val)
			nret++
		case 's':
			var size int
			size, i = parsePackSize(format, i, 8)
			if size < 1 || size > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
			}
			val := unpackSizedString(data, &offset, fs, size)
			v.Set(nret, val)
			nret++
		case 'x':
			applyAlignPad(&offset, fs, 1)
			if offset+1 > len(data) {
				panic("bad argument #2 to 'string.unpack' (data string too short)")
			}
			offset++
		case 'X':
			natAlign := getXAlignUnpack(format, &i)
			align := fs.effectiveAlign(natAlign)
			pad := addPadding(offset, align)
			offset += pad
		default:
			panic(fmt.Sprintf("invalid format option '%c'", ch))
		}
	}

	// Return values + next position (1-based)
	v.Set(nret, vm.NewInt(int64(offset+1)))
	return nret + 1
}

// getXAlignUnpack parses X directive for unpack (same logic, different error source).
func getXAlignUnpack(format string, i *int) int {
	if *i >= len(format) {
		panic("bad argument #1 to 'string.unpack' (invalid next option for option 'X')")
	}
	ch := format[*i]
	*i++
	switch ch {
	case 'b', 'B':
		return 1
	case 'h', 'H':
		return 2
	case 'l', 'L', 'j', 'J', 'T':
		return 8
	case 'i', 'I':
		var size int
		size, *i = parsePackSize(format, *i, 4)
		if size < 1 || size > 16 {
			panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
		}
		return size
	case 'f':
		return 4
	case 'd', 'n':
		return 8
	case 'x':
		return 1
	case 's':
		var size int
		size, *i = parsePackSize(format, *i, 8)
		if size < 1 || size > 16 {
			panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
		}
		return size
	case 'c', 'z', '<', '>', '=', '!', 'X':
		panic("bad argument #1 to 'string.unpack' (invalid next option for option 'X')")
	case ' ':
		for *i < len(format) && format[*i] == ' ' {
			*i++
		}
		panic("bad argument #1 to 'string.unpack' (invalid next option for option 'X')")
	default:
		panic(fmt.Sprintf("invalid format option '%c'", ch))
	}
}

func applyAlignPad(offset *int, fs *formatState, natAlign int) {
	align := fs.effectiveAlign(natAlign)
	pad := addPadding(*offset, align)
	*offset += pad
}

func unpackInt(data string, offset *int, fs *formatState, kind byte, size int, signed bool) vm.Value {
	natAlign := getNatAlign(kind, size)
	align := fs.effectiveAlign(natAlign)
	pad := addPadding(*offset, align)
	*offset += pad

	if *offset+size > len(data) {
		panic("bad argument #2 to 'string.unpack' (data string too short)")
	}

	raw := []byte(data[*offset : *offset+size])
	*offset += size

	if size <= 8 {
		var u uint64
		if fs.byteOrder == binary.LittleEndian {
			for j := size - 1; j >= 0; j-- {
				u = (u << 8) | uint64(raw[j])
			}
		} else {
			for j := 0; j < size; j++ {
				u = (u << 8) | uint64(raw[j])
			}
		}
		if signed && size < 8 {
			// Sign extend
			signBit := uint64(1) << (uint(size)*8 - 1)
			if u&signBit != 0 {
				u |= ^((uint64(1) << (uint(size) * 8)) - 1)
			}
		}
		return vm.NewInt(int64(u))
	}

	// size > 8: read 8 low bytes + check extension bytes
	var val int64
	if fs.byteOrder == binary.LittleEndian {
		val = int64(binary.LittleEndian.Uint64(raw[:8]))
		// Check extension bytes
		var ext byte
		if val < 0 {
			ext = 0xFF
		}
		for j := 8; j < size; j++ {
			if signed {
				if raw[j] != ext {
					panic(fmt.Sprintf("%d-byte integer does not fit into Lua Integer", size))
				}
			} else if raw[j] != 0 {
				// Unsigned: every surplus high byte must be zero (Lua 5.5
				// unpackint uses mask=0 for the unsigned case).
				panic(fmt.Sprintf("%d-byte integer does not fit into Lua Integer", size))
			}
		}
	} else {
		// Big endian: extension bytes are first
		var ext byte
		// Read the 8 value bytes (last 8)
		val = int64(binary.BigEndian.Uint64(raw[size-8:]))
		if val < 0 {
			ext = 0xFF
		}
		for j := 0; j < size-8; j++ {
			if signed {
				if raw[j] != ext {
					panic(fmt.Sprintf("%d-byte integer does not fit into Lua Integer", size))
				}
			} else {
				if raw[j] != 0 {
					panic(fmt.Sprintf("%d-byte integer does not fit into Lua Integer", size))
				}
			}
		}
	}
	return vm.NewInt(val)
}

func unpackFloat32(data string, offset *int, fs *formatState) vm.Value {
	natAlign := 4
	align := fs.effectiveAlign(natAlign)
	pad := addPadding(*offset, align)
	*offset += pad

	if *offset+4 > len(data) {
		panic("bad argument #2 to 'string.unpack' (data string too short)")
	}

	raw := []byte(data[*offset : *offset+4])
	*offset += 4

	var bits uint32
	if fs.byteOrder == binary.LittleEndian {
		bits = binary.LittleEndian.Uint32(raw)
	} else {
		bits = binary.BigEndian.Uint32(raw)
	}
	f := math.Float32frombits(bits)
	return vm.NewFloat(float64(f))
}

func unpackFloat64(data string, offset *int, fs *formatState, kind byte) vm.Value {
	natAlign := 8
	align := fs.effectiveAlign(natAlign)
	pad := addPadding(*offset, align)
	*offset += pad

	if *offset+8 > len(data) {
		panic("bad argument #2 to 'string.unpack' (data string too short)")
	}

	raw := []byte(data[*offset : *offset+8])
	*offset += 8

	var bits uint64
	if fs.byteOrder == binary.LittleEndian {
		bits = binary.LittleEndian.Uint64(raw)
	} else {
		bits = binary.BigEndian.Uint64(raw)
	}
	f := math.Float64frombits(bits)
	return vm.NewFloat(f)
}

func unpackFixedString(data string, offset *int, size int) vm.Value {
	if *offset+size > len(data) {
		panic("bad argument #2 to 'string.unpack' (data string too short)")
	}
	s := data[*offset : *offset+size]
	*offset += size
	return vm.NewString(s)
}

func unpackZeroTermString(data string, offset *int) vm.Value {
	idx := strings.IndexByte(data[*offset:], 0)
	if idx < 0 {
		panic("bad argument #2 to 'string.unpack' (unfinished string for format 'z')")
	}
	s := data[*offset : *offset+idx]
	*offset += idx + 1 // skip the null terminator
	return vm.NewString(s)
}

func unpackSizedString(data string, offset *int, fs *formatState, prefixSize int) vm.Value {
	align := fs.effectiveAlign(prefixSize)
	pad := addPadding(*offset, align)
	*offset += pad

	if *offset+prefixSize > len(data) {
		panic("bad argument #2 to 'string.unpack' (data string too short)")
	}

	// Read length
	raw := []byte(data[*offset : *offset+prefixSize])
	*offset += prefixSize

	var slen uint64
	if prefixSize <= 8 {
		if fs.byteOrder == binary.LittleEndian {
			for j := prefixSize - 1; j >= 0; j-- {
				slen = (slen << 8) | uint64(raw[j])
			}
		} else {
			for j := 0; j < prefixSize; j++ {
				slen = (slen << 8) | uint64(raw[j])
			}
		}
	} else {
		if fs.byteOrder == binary.LittleEndian {
			slen = binary.LittleEndian.Uint64(raw[:8])
			for j := 8; j < prefixSize; j++ {
				if raw[j] != 0 {
					panic(fmt.Sprintf("%d-byte integer does not fit into Lua Integer", prefixSize))
				}
			}
		} else {
			for j := 0; j < prefixSize-8; j++ {
				if raw[j] != 0 {
					panic(fmt.Sprintf("%d-byte integer does not fit into Lua Integer", prefixSize))
				}
			}
			slen = binary.BigEndian.Uint64(raw[prefixSize-8:])
		}
	}

	// Compare in uint64 before int conversion: when slen has its top bit set,
	// int(slen) overflows negative on 64-bit platforms, and the bounds check
	// becomes vacuously true while the slice operation panics with a Go
	// "slice bounds out of range" runtime error that leaks through pcall.
	if slen > uint64(len(data)-*offset) {
		panic("bad argument #2 to 'string.unpack' (data string too short)")
	}

	s := data[*offset : *offset+int(slen)]
	*offset += int(slen)
	return vm.NewString(s)
}

// stringPacksize implements string.packsize(fmt)
func stringPacksize(v *vm.VM) int {
	format := getString(v, 1, "string.packsize")
	fs := newFormatState("string.packsize")
	totalSize := 0

	i := 0
	for i < len(format) {
		ch := format[i]
		i++

		switch ch {
		case ' ':
			continue
		case '<', '>', '=':
			continue
		case '!':
			var align int
			align, i = parsePackSize(format, i, 8)
			if align < 1 || align > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", align))
			}
			fs.maxAlign = align
			continue
		}

		switch ch {
		case 'b', 'B':
			totalSize += addPadding(totalSize, fs.effectiveAlign(1))
			totalSize += 1
		case 'h', 'H':
			totalSize += addPadding(totalSize, fs.effectiveAlign(2))
			totalSize += 2
		case 'l', 'L', 'j', 'J', 'T':
			totalSize += addPadding(totalSize, fs.effectiveAlign(8))
			totalSize += 8
		case 'i', 'I':
			var size int
			size, i = parsePackSize(format, i, 4)
			if size < 1 || size > 16 {
				panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
			}
			totalSize += addPadding(totalSize, fs.effectiveAlign(size))
			totalSize += size
		case 'f':
			totalSize += addPadding(totalSize, fs.effectiveAlign(4))
			totalSize += 4
		case 'd', 'n':
			totalSize += addPadding(totalSize, fs.effectiveAlign(8))
			totalSize += 8
		case 'c':
			if i >= len(format) || format[i] < '0' || format[i] > '9' {
				panic("missing size for format option 'c'")
			}
			var size int
			size, i = parsePackSize(format, i, 0)
			totalSize += size
		case 'x':
			totalSize += addPadding(totalSize, fs.effectiveAlign(1))
			totalSize += 1
		case 'X':
			natAlign := getXAlignPacksize(format, &i)
			align := fs.effectiveAlign(natAlign)
			totalSize += addPadding(totalSize, align)
		case 'z', 's':
			panic("bad argument #1 to 'string.packsize' (variable-length format)")
		default:
			panic(fmt.Sprintf("invalid format option '%c'", ch))
		}

		if totalSize > maxPackSize {
			panic("bad argument #1 to 'string.packsize' (format result too large)")
		}
	}

	v.Set(0, vm.NewInt(int64(totalSize)))
	return 1
}

// getXAlignPacksize parses X directive for packsize.
func getXAlignPacksize(format string, i *int) int {
	if *i >= len(format) {
		panic("bad argument #1 to 'string.packsize' (invalid next option for option 'X')")
	}
	ch := format[*i]
	*i++
	switch ch {
	case 'b', 'B':
		return 1
	case 'h', 'H':
		return 2
	case 'l', 'L', 'j', 'J', 'T':
		return 8
	case 'i', 'I':
		var size int
		size, *i = parsePackSize(format, *i, 4)
		if size < 1 || size > 16 {
			panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
		}
		return size
	case 'f':
		return 4
	case 'd', 'n':
		return 8
	case 'x':
		return 1
	case 's':
		var size int
		size, *i = parsePackSize(format, *i, 8)
		if size < 1 || size > 16 {
			panic(fmt.Sprintf("integral size (%d) out of limits [1,16]", size))
		}
		return size
	case 'c', 'z', '<', '>', '=', '!', 'X':
		panic("bad argument #1 to 'string.packsize' (invalid next option for option 'X')")
	case ' ':
		for *i < len(format) && format[*i] == ' ' {
			*i++
		}
		panic("bad argument #1 to 'string.packsize' (invalid next option for option 'X')")
	default:
		panic(fmt.Sprintf("invalid format option '%c'", ch))
	}
}
