package compiler

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// putVarint encodes v in the Lua 5.4 variable-length unsigned format the
// undumper reads: 7 data bits per byte, most-significant group first, with the
// high bit (0x80) set on the final byte.
func putVarint(buf *bytes.Buffer, v uint64) {
	var groups []byte
	for {
		groups = append([]byte{byte(v & 0x7f)}, groups...)
		v >>= 7
		if v == 0 {
			break
		}
	}
	groups[len(groups)-1] |= 0x80
	buf.Write(groups)
}

// validHeader writes the exact 5.4 binary-chunk header (through the top-level
// upvalue count) that checkHeader accepts, so the bytes appended afterward are
// parsed by loadFunction.
func validHeader(buf *bytes.Buffer) {
	buf.WriteString("\x1bLua")
	buf.WriteByte(0x54) // version
	buf.WriteByte(0x00) // format
	buf.WriteString("\x19\x93\r\n\x1a\n")
	buf.WriteByte(4) // Instruction size
	buf.WriteByte(8) // lua_Integer size
	buf.WriteByte(8) // lua_Number size
	_ = binary.Write(buf, binary.LittleEndian, int64(0x5678))
	_ = binary.Write(buf, binary.LittleEndian, float64(370.5))
	buf.WriteByte(0x00) // top-level upvalue count
}

// loadFunctionPrefix writes the leading fields of a function body up to (but
// not including) the first element count, so a test can place a hostile count
// next.
func loadFunctionPrefix(buf *bytes.Buffer) {
	putVarint(buf, 0) // source: size==0 -> empty string
	putVarint(buf, 0) // LineDef
	putVarint(buf, 0) // LastLine
	buf.WriteByte(0)  // NumParams
	buf.WriteByte(0)  // IsVarArg
	buf.WriteByte(0)  // MaxStack
}

// TestUndumpHugeCountIsCatchable is a regression guard for the sandbox-escape
// where a corrupt element count in a binary chunk drove make([]T, count) into
// an uncatchable Go fatal OOM (recover() cannot catch runtime.throw OOM), so a
// tiny crafted string passed to load() crashed the host. Every count that
// drives an allocation must now be bounded by the bytes remaining in the chunk
// and surface as an ordinary "bad binary format" error.
//
// Pre-fix this test would abort the whole test binary with a multi-GB OOM;
// post-fix Undump returns a normal error.
func TestUndumpHugeCountIsCatchable(t *testing.T) {
	const huge = 2_000_000_000 // ~8GB if it reached make([]Instruction, huge)

	var buf bytes.Buffer
	validHeader(&buf)
	loadFunctionPrefix(&buf)
	putVarint(&buf, huge) // nCode: far exceeds the (zero) remaining bytes

	proto, _, err := Undump(buf.Bytes(), "evil")
	if err == nil {
		t.Fatalf("expected a catchable error for an inflated code count, got proto=%v", proto)
	}
	if proto != nil {
		t.Fatalf("expected nil proto on error, got %v", proto)
	}
}
