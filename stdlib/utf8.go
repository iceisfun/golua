package stdlib

import (
	"unicode/utf8"

	"github.com/iceisfun/golua/vm"
)

func openUtf8(v *vm.VM) {
	u := vm.NewEmptyTable()

	u.SetString("char", vm.NewNativeFunc(luaUtf8Char))
	u.SetString("codepoint", vm.NewNativeFunc(luaUtf8Codepoint))
	u.SetString("codes", vm.NewNativeFunc(luaUtf8Codes))
	u.SetString("len", vm.NewNativeFunc(luaUtf8Len))
	u.SetString("offset", vm.NewNativeFunc(luaUtf8Offset))

	// utf8.charpattern: matches exactly one UTF-8 byte sequence
	// Range \xC2-\xFD covers lead bytes for 2-byte through 6-byte sequences,
	// matching Lua 5.4's extended UTF-8 support.
	u.SetString("charpattern", vm.NewString("[\x00-\x7F\xC2-\xFD][\x80-\xBF]*"))

	v.SetGlobal("utf8", vm.NewTable(u))
}

// utf8.char(...) — encode codepoints to UTF-8 string
// Lua 5.4 allows the full extended range 0x00–0x7FFFFFFF (up to 6-byte sequences).
func luaUtf8Char(v *vm.VM) int {
	n := v.ArgCount()
	if n == 0 {
		v.Set(0, vm.NewString(""))
		return 1
	}

	var buf []byte
	for i := 1; i <= n; i++ {
		val := v.Get(i)
		code, ok := val.ToInt()
		if !ok {
			callerArgError(v, i, "utf8.char", "number has no integer representation")
		}
		if code < 0 || code > 0x7FFFFFFF {
			callerArgError(v, i, "utf8.char", "value out of range")
		}
		buf = appendExtendedUTF8(buf, uint32(code))
	}
	v.Set(0, vm.NewString(string(buf)))
	return 1
}

// appendExtendedUTF8 encodes a codepoint as extended UTF-8 (up to 6 bytes),
// supporting Lua's full range of 0x00–0x7FFFFFFF.
func appendExtendedUTF8(buf []byte, cp uint32) []byte {
	switch {
	case cp <= 0x7F:
		return append(buf, byte(cp))
	case cp <= 0x7FF:
		return append(buf, byte(0xC0|(cp>>6)), byte(0x80|(cp&0x3F)))
	case cp <= 0xFFFF:
		return append(buf, byte(0xE0|(cp>>12)), byte(0x80|((cp>>6)&0x3F)), byte(0x80|(cp&0x3F)))
	case cp <= 0x1FFFFF:
		return append(buf, byte(0xF0|(cp>>18)), byte(0x80|((cp>>12)&0x3F)), byte(0x80|((cp>>6)&0x3F)), byte(0x80|(cp&0x3F)))
	case cp <= 0x3FFFFFF:
		return append(buf, byte(0xF8|(cp>>24)), byte(0x80|((cp>>18)&0x3F)), byte(0x80|((cp>>12)&0x3F)), byte(0x80|((cp>>6)&0x3F)), byte(0x80|(cp&0x3F)))
	default: // up to 0x7FFFFFFF
		return append(buf, byte(0xFC|(cp>>30)), byte(0x80|((cp>>24)&0x3F)), byte(0x80|((cp>>18)&0x3F)), byte(0x80|((cp>>12)&0x3F)), byte(0x80|((cp>>6)&0x3F)), byte(0x80|(cp&0x3F)))
	}
}

// decodeExtendedUTF8 decodes one extended UTF-8 character (up to 6 bytes).
// Returns the codepoint and byte count, or (-1, 1) on error.
func decodeExtendedUTF8(s string) (int64, int) {
	if len(s) == 0 {
		return -1, 0
	}
	b := s[0]
	switch {
	case b <= 0x7F:
		return int64(b), 1
	case b <= 0xBF:
		return -1, 1 // continuation byte
	case b <= 0xDF:
		if len(s) < 2 || s[1]&0xC0 != 0x80 {
			return -1, 1
		}
		cp := int64(b&0x1F)<<6 | int64(s[1]&0x3F)
		if cp < 0x80 {
			return -1, 1
		}
		return cp, 2
	case b <= 0xEF:
		if len(s) < 3 {
			return -1, 1
		}
		for j := 1; j < 3; j++ {
			if s[j]&0xC0 != 0x80 {
				return -1, 1
			}
		}
		cp := int64(b&0x0F)<<12 | int64(s[1]&0x3F)<<6 | int64(s[2]&0x3F)
		if cp < 0x800 {
			return -1, 1
		}
		return cp, 3
	case b <= 0xF7:
		if len(s) < 4 {
			return -1, 1
		}
		for j := 1; j < 4; j++ {
			if s[j]&0xC0 != 0x80 {
				return -1, 1
			}
		}
		cp := int64(b&0x07)<<18 | int64(s[1]&0x3F)<<12 | int64(s[2]&0x3F)<<6 | int64(s[3]&0x3F)
		if cp < 0x10000 {
			return -1, 1
		}
		return cp, 4
	case b <= 0xFB:
		if len(s) < 5 {
			return -1, 1
		}
		for j := 1; j < 5; j++ {
			if s[j]&0xC0 != 0x80 {
				return -1, 1
			}
		}
		cp := int64(b&0x03)<<24 | int64(s[1]&0x3F)<<18 | int64(s[2]&0x3F)<<12 | int64(s[3]&0x3F)<<6 | int64(s[4]&0x3F)
		if cp < 0x200000 {
			return -1, 1
		}
		return cp, 5
	case b <= 0xFD:
		if len(s) < 6 {
			return -1, 1
		}
		for j := 1; j < 6; j++ {
			if s[j]&0xC0 != 0x80 {
				return -1, 1
			}
		}
		cp := int64(b&0x01)<<30 | int64(s[1]&0x3F)<<24 | int64(s[2]&0x3F)<<18 | int64(s[3]&0x3F)<<12 | int64(s[4]&0x3F)<<6 | int64(s[5]&0x3F)
		if cp < 0x4000000 {
			return -1, 1
		}
		return cp, 6
	default:
		return -1, 1
	}
}

// utf8.len(s [, i [, j [, lax]]]) — count UTF-8 characters; soft fail on invalid
func luaUtf8Len(v *vm.VM) int {
	s := getString(v, 1, "utf8.len")
	slen := len(s)

	// Default: i=1, j=-1
	posi := int64(1)
	if !v.Get(2).IsNil() {
		posi = getInt(v, 2, "utf8.len")
	}
	posj := int64(-1)
	if !v.Get(3).IsNil() {
		posj = getInt(v, 3, "utf8.len")
	}

	// Lax mode (arg 4): when true, accept extended codepoints (> U+10FFFF)
	lax := v.Get(4).ToBool()

	// Resolve relative positions (1-indexed)
	i := posRelat(posi, slen)
	j := posRelat(posj, slen)

	// Validate bounds (Lua: 1 <= i <= #s+1, j <= #s)
	if i < 1 || i > slen+1 {
		callerArgError(v, 2, "utf8.len", "initial position out of bounds")
	}
	if j > slen {
		callerArgError(v, 3, "utf8.len", "final position out of bounds")
	}

	// Convert to 0-indexed byte offsets
	start := i - 1
	end := j // j is inclusive in Lua, so end = j (0-indexed: j-1+1)

	count := int64(0)
	p := start
	for p < end {
		if lax {
			cp, size := decodeExtendedUTF8(s[p:])
			if cp < 0 || size == 0 {
				v.Set(0, vm.Nil)
				v.Set(1, vm.NewInt(int64(p+1)))
				return 2
			}
			p += size
		} else {
			r, size := utf8.DecodeRuneInString(s[p:])
			if r == utf8.RuneError && size <= 1 {
				// Invalid UTF-8: return nil, byte position (1-indexed)
				v.Set(0, vm.Nil)
				v.Set(1, vm.NewInt(int64(p+1)))
				return 2
			}
			p += size
		}
		count++
	}

	v.Set(0, vm.NewInt(count))
	return 1
}

// utf8.codepoint(s [, i [, j [, lax]]]) — return codepoints as integers
func luaUtf8Codepoint(v *vm.VM) int {
	s := getString(v, 1, "utf8.codepoint")
	slen := len(s)

	// Default: i=1, j=i
	posi := int64(1)
	if !v.Get(2).IsNil() {
		posi = getInt(v, 2, "utf8.codepoint")
	}
	posj := posi
	if !v.Get(3).IsNil() {
		posj = getInt(v, 3, "utf8.codepoint")
	}

	// Lax mode (arg 4): when true, accept extended codepoints (surrogates, > U+10FFFF)
	lax := v.Get(4).ToBool()

	// Resolve relative positions
	i := posRelat(posi, slen)
	j := posRelat(posj, slen)

	if i < 1 {
		callerArgError(v, 2, "utf8.codepoint", "out of bounds")
	}
	if j > slen {
		callerArgError(v, 3, "utf8.codepoint", "out of bounds")
	}
	if i > j {
		return 0 // empty interval
	}

	// Convert to 0-indexed
	start := i - 1
	end := j // inclusive end in 0-indexed = j-1+1 = j

	// Upper-bound estimate: at most (end - start) codepoints
	v.EnsureStack(v.Base() + (end - start))

	n := 0
	for p := start; p < end; {
		if lax {
			cp, size := decodeExtendedUTF8(s[p:])
			if cp < 0 || size == 0 {
				panic("invalid UTF-8 code")
			}
			v.Set(n, vm.NewInt(cp))
			n++
			p += size
		} else {
			r, size := utf8.DecodeRuneInString(s[p:])
			if r == utf8.RuneError && size <= 1 {
				panic("invalid UTF-8 code")
			}
			v.Set(n, vm.NewInt(int64(r)))
			n++
			p += size
		}
	}
	return n
}

// utf8.codes(s [, lax]) — iterator factory
func luaUtf8Codes(v *vm.VM) int {
	s := getString(v, 1, "utf8.codes")

	// Lax mode (arg 2): when true, accept extended codepoints (> U+10FFFF)
	lax := v.Get(2).ToBool()
	if !lax && len(s) > 0 && !utf8.RuneStart(s[0]) {
		callerArgError(v, 1, "utf8.codes", "invalid UTF-8 code")
	}

	// Return iterator, string, initial state (0)
	iter := vm.NewNativeFunc(func(v *vm.VM) int {
		str := v.Get(1).AsString()
		state, _ := v.Get(2).ToInt()

		// State is: 0 = initial, >0 = 1-indexed position of last returned codepoint.
		// Convert to 0-indexed byte offset for next codepoint to decode.
		n := int(state)

		if n < 0 {
			v.Set(0, vm.Nil)
			return 1
		}

		if n == 0 {
			// Initial call: decode at byte 0
			n = 0
		} else {
			// Advance past previous codepoint: state is 1-indexed position,
			// so byte[state-1] was the start of the previous codepoint.
			// Decode it to find its size, then advance past it.
			prev := n - 1 // 0-indexed position of previous codepoint
			if prev < len(str) {
				if lax {
					_, size := decodeExtendedUTF8(str[prev:])
					if size > 0 {
						n = prev + size
					}
				} else {
					_, size := utf8.DecodeRuneInString(str[prev:])
					n = prev + size
				}
			}
		}

		if n >= len(str) {
			v.Set(0, vm.Nil)
			return 1
		}

		if lax {
			cp, size := decodeExtendedUTF8(str[n:])
			if cp < 0 || size == 0 {
				panic("invalid UTF-8 code")
			}
			v.Set(0, vm.NewInt(int64(n+1)))
			v.Set(1, vm.NewInt(cp))
			return 2
		}

		// In strict mode, stray continuation bytes are invalid.
		if !utf8.RuneStart(str[n]) {
			panic("invalid UTF-8 code")
		}

		r, size := utf8.DecodeRuneInString(str[n:])
		if r == utf8.RuneError && size <= 1 {
			panic("invalid UTF-8 code")
		}

		// Return 1-indexed position and codepoint
		v.Set(0, vm.NewInt(int64(n+1)))
		v.Set(1, vm.NewInt(int64(r)))
		return 2
	})

	v.Set(0, iter)
	v.Set(1, vm.NewString(s))
	v.Set(2, vm.NewInt(0))
	return 3
}

// utf8.offset(s, n [, i]) — convert codepoint offset to byte position
func luaUtf8Offset(v *vm.VM) int {
	s := getString(v, 1, "utf8.offset")
	n := getInt(v, 2, "utf8.offset")
	slen := len(s)

	// Default for i depends on n
	var posi int64
	if n >= 0 {
		posi = 1
	} else {
		posi = int64(slen) + 1
	}
	if !v.Get(3).IsNil() {
		posi = getInt(v, 3, "utf8.offset")
	}

	// Resolve relative position
	i := posRelat(posi, slen)

	// Validate: 1 <= i <= #s+1 (Lua C: 1 <= posi && --posi <= len)
	if i < 1 || i > slen+1 {
		callerArgError(v, 3, "utf8.offset", "position out of bounds")
	}

	// Convert to 0-indexed
	p := i - 1

	if n == 0 {
		if p == slen {
			v.Set(0, vm.NewInt(int64(slen+1)))
			return 1
		}
		// Find beginning of current character by walking backwards
		for p > 0 && !utf8.RuneStart(s[p]) {
			p--
		}
	} else if n > 0 {
		// Check not a continuation byte
		if p < slen && !utf8.RuneStart(s[p]) {
			panic("initial position is a continuation byte")
		}
		// n=1 means "character at p", so decrement first
		n--
		for n > 0 && p < slen {
			p++
			for p < slen && !utf8.RuneStart(s[p]) {
				p++
			}
			n--
		}
	} else {
		// n < 0: walk backwards
		if p < slen && !utf8.RuneStart(s[p]) {
			panic("initial position is a continuation byte")
		}
		for n < 0 && p > 0 {
			p--
			for p > 0 && !utf8.RuneStart(s[p]) {
				p--
			}
			n++
		}
	}

	if n == 0 {
		// Found the target character
		v.Set(0, vm.NewInt(int64(p+1))) // 1-indexed
	} else {
		// Could not traverse n characters
		v.Set(0, vm.Nil)
	}
	return 1
}
