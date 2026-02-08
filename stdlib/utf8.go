package stdlib

import (
	"fmt"
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
	u.SetString("charpattern", vm.NewString("[\x00-\x7F\xC2-\xF4][\x80-\xBF]*"))

	v.SetGlobal("utf8", vm.NewTable(u))
}

// utf8.char(...) — encode codepoints to UTF-8 string
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
			panic(fmt.Sprintf("bad argument #%d to 'char' (number has no integer representation)", i))
		}
		r := rune(code)
		if !utf8.ValidRune(r) {
			panic(fmt.Sprintf("bad argument #%d to 'char' (value out of range)", i))
		}
		buf = utf8.AppendRune(buf, r)
	}
	v.Set(0, vm.NewString(string(buf)))
	return 1
}

// utf8.len(s [, i [, j]]) — count UTF-8 characters; soft fail on invalid
func luaUtf8Len(v *vm.VM) int {
	s := getString(v, 1, "len")
	slen := len(s)

	// Default: i=1, j=-1
	posi := int64(1)
	if !v.Get(2).IsNil() {
		posi = getInt(v, 2, "len")
	}
	posj := int64(-1)
	if !v.Get(3).IsNil() {
		posj = getInt(v, 3, "len")
	}

	// Accept lax parameter (arg 4) for API compatibility with Lua 5.4.
	// golua uses Go's unicode/utf8 for all decoding, so lax mode behaves
	// identically to strict mode on valid UTF-8. Invalid sequences still
	// error regardless of the flag.

	// Resolve relative positions (1-indexed)
	i := posRelat(posi, slen)
	j := posRelat(posj, slen)

	// Validate bounds (Lua: 1 <= i <= #s+1, j <= #s)
	if i < 1 || i > slen+1 {
		panic("bad argument #2 to 'len' (initial position out of bounds)")
	}
	if j > slen {
		panic("bad argument #3 to 'len' (final position out of bounds)")
	}

	// Convert to 0-indexed byte offsets
	start := i - 1
	end := j // j is inclusive in Lua, so end = j (0-indexed: j-1+1)

	count := int64(0)
	p := start
	for p < end {
		r, size := utf8.DecodeRuneInString(s[p:])
		if r == utf8.RuneError && size <= 1 {
			// Invalid UTF-8: return nil, byte position (1-indexed)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewInt(int64(p+1)))
			return 2
		}
		p += size
		count++
	}

	v.Set(0, vm.NewInt(count))
	return 1
}

// utf8.codepoint(s [, i [, j]]) — return codepoints as integers
func luaUtf8Codepoint(v *vm.VM) int {
	s := getString(v, 1, "codepoint")
	slen := len(s)

	// Default: i=1, j=i
	posi := int64(1)
	if !v.Get(2).IsNil() {
		posi = getInt(v, 2, "codepoint")
	}
	posj := posi
	if !v.Get(3).IsNil() {
		posj = getInt(v, 3, "codepoint")
	}

	// Accept lax parameter (arg 4) for Lua 5.4 API compatibility.

	// Resolve relative positions
	i := posRelat(posi, slen)
	j := posRelat(posj, slen)

	if i < 1 {
		panic("bad argument #2 to 'codepoint' (out of bounds)")
	}
	if j > slen {
		panic("bad argument #3 to 'codepoint' (out of bounds)")
	}
	if i > j {
		return 0 // empty interval
	}

	// Convert to 0-indexed
	start := i - 1
	end := j // inclusive end in 0-indexed = j-1+1 = j

	n := 0
	for p := start; p < end; {
		r, size := utf8.DecodeRuneInString(s[p:])
		if r == utf8.RuneError && size <= 1 {
			panic("invalid UTF-8 code")
		}
		v.Set(n, vm.NewInt(int64(r)))
		n++
		p += size
	}
	return n
}

// utf8.codes(s) — iterator factory
func luaUtf8Codes(v *vm.VM) int {
	s := getString(v, 1, "codes")

	// Accept lax parameter (arg 2) for Lua 5.4 API compatibility.

	// Validate first byte is not a continuation byte
	if len(s) > 0 && !utf8.RuneStart(s[0]) {
		panic("bad argument #1 to 'codes' (invalid UTF-8 code)")
	}

	// Return iterator, string, initial state (0)
	iter := vm.NewNativeFunc(func(v *vm.VM) int {
		str := v.Get(1).AsString()
		state, _ := v.Get(2).ToInt()

		// Convert state to byte offset (0-indexed)
		n := int(state)

		// Skip continuation bytes (handles post-decode position)
		for n < len(str) && !utf8.RuneStart(str[n]) {
			n++
		}

		if n >= len(str) {
			v.Set(0, vm.Nil)
			return 1
		}

		r, size := utf8.DecodeRuneInString(str[n:])
		if r == utf8.RuneError && size <= 1 {
			panic("invalid UTF-8 code")
		}
		// Check that the byte after the decoded rune is not a continuation byte
		// (this catches overlong/invalid sequences that DecodeRune may skip)
		next := n + size
		if next < len(str) && !utf8.RuneStart(str[next]) {
			// Verify by attempting to decode from next position
			r2, s2 := utf8.DecodeRuneInString(str[next:])
			if r2 == utf8.RuneError && s2 <= 1 {
				panic("invalid UTF-8 code")
			}
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
	s := getString(v, 1, "offset")
	n := getInt(v, 2, "offset")
	slen := len(s)

	// Default for i depends on n
	var posi int64
	if n >= 0 {
		posi = 1
	} else {
		posi = int64(slen) + 1
	}
	if !v.Get(3).IsNil() {
		posi = getInt(v, 3, "offset")
	}

	// Resolve relative position
	i := posRelat(posi, slen)

	// Validate: 1 <= i <= #s+1 (Lua C: 1 <= posi && --posi <= len)
	if i < 1 || i > slen+1 {
		panic("bad argument #3 to 'offset' (position out of bounds)")
	}

	// Convert to 0-indexed
	p := i - 1

	if n == 0 {
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
