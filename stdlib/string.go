package stdlib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/vm"
)

func openString(v *vm.VM) {
	str := vm.NewEmptyTable()

	str.SetString("len", vm.NewNativeFunc(stringLen))
	str.SetString("sub", vm.NewNativeFunc(stringSub))
	str.SetString("upper", vm.NewNativeFunc(stringUpper))
	str.SetString("lower", vm.NewNativeFunc(stringLower))
	str.SetString("rep", vm.NewNativeFunc(stringRep))
	str.SetString("reverse", vm.NewNativeFunc(stringReverse))
	str.SetString("byte", vm.NewNativeFunc(stringByte))
	str.SetString("char", vm.NewNativeFunc(stringChar))
	str.SetString("find", vm.NewNativeFunc(stringFind))
	str.SetString("format", vm.NewNativeFunc(stringFormat))
	str.SetString("gsub", vm.NewNativeFunc(stringGsub))
	str.SetString("match", vm.NewNativeFunc(stringMatch))
	str.SetString("gmatch", vm.NewNativeFunc(stringGmatch))
	str.SetString("dump", vm.NewNativeFunc(stringDump))

	v.SetGlobal("string", vm.NewTable(str))

	// Set string metatable so strings can use method syntax (str:find(...))
	// The string table itself serves as the __index for strings
	v.SetStringMeta(str)
}

// string.len(s)
func stringLen(v *vm.VM) int {
	s := getString(v, 1, "len")
	v.Set(0, vm.NewInt(int64(len(s))))
	return 1
}

// string.sub(s, i [, j])
func stringSub(v *vm.VM) int {
	s := getString(v, 1, "sub")
	i := getInt(v, 2, "sub")
	j := int64(len(s))
	if !v.Get(3).IsNil() {
		j = getInt(v, 3, "sub")
	}

	// Lua uses 1-based indexing, negative indices count from end
	start := posRelat(i, len(s))
	end := posRelat(j, len(s))

	if start < 1 {
		start = 1
	}
	if end > len(s) {
		end = len(s)
	}
	if start > end {
		v.Set(0, vm.NewString(""))
		return 1
	}

	v.Set(0, vm.NewString(s[start-1:end]))
	return 1
}

// string.upper(s) — ASCII-only, byte-level (Lua strings are byte sequences)
func stringUpper(v *vm.VM) int {
	s := getString(v, 1, "upper")
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	v.Set(0, vm.NewString(string(b)))
	return 1
}

// string.lower(s) — ASCII-only, byte-level (Lua strings are byte sequences)
func stringLower(v *vm.VM) int {
	s := getString(v, 1, "lower")
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	v.Set(0, vm.NewString(string(b)))
	return 1
}

// string.rep(s, n [, sep])
func stringRep(v *vm.VM) int {
	s := getString(v, 1, "rep")
	n := getInt(v, 2, "rep")
	sep := ""
	if !v.Get(3).IsNil() {
		sep = getString(v, 3, "rep")
	}

	if n <= 0 {
		v.Set(0, vm.NewString(""))
		return 1
	}

	var result strings.Builder
	for i := int64(0); i < n; i++ {
		if i > 0 {
			result.WriteString(sep)
		}
		result.WriteString(s)
	}
	v.Set(0, vm.NewString(result.String()))
	return 1
}

// string.reverse(s)
func stringReverse(v *vm.VM) int {
	s := getString(v, 1, "reverse")
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	v.Set(0, vm.NewString(string(b)))
	return 1
}

// string.byte(s [, i [, j]])
func stringByte(v *vm.VM) int {
	s := getString(v, 1, "byte")
	i := int64(1)
	if !v.Get(2).IsNil() {
		i = getInt(v, 2, "byte")
	}
	j := i
	if !v.Get(3).IsNil() {
		j = getInt(v, 3, "byte")
	}

	start := posRelat(i, len(s))
	end := posRelat(j, len(s))

	if start < 1 {
		start = 1
	}
	if end > len(s) {
		end = len(s)
	}

	n := end - start + 1
	if n > 0 {
		v.EnsureStack(v.Base() + n)
	}
	count := 0
	for idx := start; idx <= end; idx++ {
		v.Set(count, vm.NewInt(int64(s[idx-1])))
		count++
	}
	return count
}

// string.char(...)
func stringChar(v *vm.VM) int {
	n := v.ArgCount()
	var buf bytes.Buffer
	for i := 1; i <= n; i++ {
		c := getInt(v, i, "char")
		if c < 0 || c > 255 {
			panic(fmt.Sprintf("bad argument #%d to 'char' (value out of range)", i))
		}
		buf.WriteByte(byte(c))
	}
	v.Set(0, vm.NewString(buf.String()))
	return 1
}

// string.find(s, pattern [, init [, plain]])
func stringFind(v *vm.VM) int {
	s := getString(v, 1, "find")
	pattern := getString(v, 2, "find")
	init := int64(1)
	if !v.Get(3).IsNil() {
		init = getInt(v, 3, "find")
	}
	plain := false
	if !v.Get(4).IsNil() {
		plain = v.Get(4).ToBool()
	}

	start := posRelat(init, len(s))
	if start < 1 {
		start = 1
	}
	if start > len(s)+1 {
		v.Set(0, vm.Nil)
		return 1
	}

	if plain {
		searchStr := s[start-1:]
		idx := strings.Index(searchStr, pattern)
		if idx == -1 {
			v.Set(0, vm.Nil)
			return 1
		}
		v.Set(0, vm.NewInt(int64(start+idx)))
		v.Set(1, vm.NewInt(int64(start+idx+len(pattern)-1)))
		return 2
	}

	// Pattern matching
	mStart, mEnd, caps, found := luaMatchFrom(s, pattern, int(init))
	if !found {
		v.Set(0, vm.Nil)
		return 1
	}

	v.Set(0, vm.NewInt(int64(mStart+1))) // 1-based
	v.Set(1, vm.NewInt(int64(mEnd)))     // inclusive end
	nret := 2

	for i, c := range caps {
		if c.isPos {
			v.Set(2+i, vm.NewInt(int64(c.pos)))
		} else {
			v.Set(2+i, vm.NewString(c.str))
		}
	}
	return nret + len(caps)
}

// string.format(formatstring, ...)
func stringFormat(v *vm.VM) int {
	format := getString(v, 1, "format")
	vals := make([]vm.Value, v.ArgCount()-1)
	for i := 2; i <= v.ArgCount(); i++ {
		vals[i-2] = v.Get(i)
	}

	result := luaFormatValues(v, format, vals)
	v.Set(0, vm.NewString(result))
	return 1
}

// string.gsub(s, pattern, repl [, n])
func stringGsub(v *vm.VM) int {
	s := getString(v, 1, "gsub")
	pattern := getString(v, 2, "gsub")
	repl := v.Get(3)
	// Validate replacement type
	if !repl.IsString() && !repl.IsFunction() && !repl.IsNativeFunc() && !repl.IsTable() {
		panic(fmt.Sprintf("bad argument #3 to 'gsub' (string/function/table expected, got %s)", repl.Type()))
	}
	maxRepl := -1
	if v.ArgCount() >= 4 && !v.Get(4).IsNil() {
		maxRepl = int(getInt(v, 4, "gsub"))
	}

	// Handle anchor
	anchored := false
	searchPat := pattern
	if len(searchPat) > 0 && searchPat[0] == '^' {
		anchored = true
		searchPat = searchPat[1:]
	}

	var result strings.Builder
	count := 0
	pos := 0          // 0-based current position
	lastMatch := -1   // 0-based end of last match, -1 = none

	for pos <= len(s) && (maxRepl < 0 || count < maxRepl) {
		end, caps, ok := luaMatchAt(s, searchPat, pos)
		if ok && end != lastMatch {
			// Valid match at [pos, end)
			hasCaps := len(caps) > 0

			// Build capture list (default to whole match if no explicit captures)
			matchCaps := caps
			if !hasCaps {
				matchCaps = []captureValue{{str: s[pos:end]}}
			}

			// Get replacement
			var replacement string
			if repl.IsString() {
				replacement = expandReplacement(repl.AsString(), s, pos, end, matchCaps)
			} else if repl.IsFunction() || repl.IsNativeFunc() {
				replacement = callGsubFunc(v, repl, matchCaps, s[pos:end])
			} else if repl.IsTable() {
				replacement = lookupGsubTable(repl, matchCaps, s[pos:end])
			}

			result.WriteString(replacement)
			count++
			lastMatch = end

			if end == pos {
				// Empty match: copy current char and advance
				if pos < len(s) {
					result.WriteByte(s[pos])
				}
				pos++
			} else {
				pos = end
			}
		} else {
			// No match or duplicate empty match: copy char and advance
			if pos < len(s) {
				result.WriteByte(s[pos])
			}
			pos++
		}

		if anchored {
			break
		}
	}

	// Append remaining text
	if pos <= len(s) {
		result.WriteString(s[pos:])
	}

	v.Set(0, vm.NewString(result.String()))
	v.Set(1, vm.NewInt(int64(count)))
	return 2
}

// expandReplacement expands a replacement string with captures.
func expandReplacement(repl string, s string, mStart, mEnd int, caps []captureValue) string {
	var result strings.Builder
	for i := 0; i < len(repl); i++ {
		if repl[i] == '%' && i+1 < len(repl) {
			next := repl[i+1]
			if next >= '0' && next <= '9' {
				idx := int(next - '0')
				if idx == 0 {
					result.WriteString(s[mStart:mEnd])
				} else if idx <= len(caps) {
					c := caps[idx-1]
					if c.isPos {
						result.WriteString(fmt.Sprintf("%d", c.pos))
					} else {
						result.WriteString(c.str)
					}
				} else {
					panic(fmt.Sprintf("invalid use of '%%%c' in replacement string", next))
				}
				i++
				continue
			} else if next == '%' {
				result.WriteByte('%')
				i++
				continue
			}
		}
		result.WriteByte(repl[i])
	}
	return result.String()
}

// callGsubFunc calls a function for gsub replacement.
func callGsubFunc(v *vm.VM, fn vm.Value, captures []captureValue, wholeMatch string) string {
	args := make([]vm.Value, len(captures))
	for i, cap := range captures {
		if cap.isPos {
			args[i] = vm.NewInt(int64(cap.pos))
		} else {
			args[i] = vm.NewString(cap.str)
		}
	}

	// Use ProtectedCall but re-panic on error (Lua propagates gsub function errors)
	results, err := v.ProtectedCall(fn, args)
	if err != nil {
		panic(err.Error())
	}
	if len(results) == 0 {
		return wholeMatch
	}

	ret := results[0]
	if ret.IsString() {
		return ret.AsString()
	} else if ret.IsNumber() {
		return valueToString(ret)
	} else if ret.IsNil() || (ret.IsBool() && !ret.AsBool()) {
		return wholeMatch
	}
	panic(fmt.Sprintf("invalid replacement value (a %s)", ret.Type()))
}

// lookupGsubTable looks up a gsub replacement from a table.
func lookupGsubTable(repl vm.Value, captures []captureValue, wholeMatch string) string {
	var key vm.Value
	c := captures[0]
	if c.isPos {
		key = vm.NewInt(int64(c.pos))
	} else {
		key = vm.NewString(c.str)
	}
	val := repl.AsTable().Get(key)
	if val.IsString() {
		return val.AsString()
	} else if val.IsNumber() {
		return valueToString(val)
	} else if val.IsNil() || (val.IsBool() && !val.AsBool()) {
		return wholeMatch
	}
	panic(fmt.Sprintf("invalid replacement value (a %s)", val.Type()))
}

// captureStr returns the string representation of a capture value.
func captureStr(c captureValue) string {
	if c.isPos {
		return fmt.Sprintf("%d", c.pos)
	}
	return c.str
}

// string.match(s, pattern [, init])
func stringMatch(v *vm.VM) int {
	s := getString(v, 1, "match")
	pattern := getString(v, 2, "match")
	init := int64(1)
	if !v.Get(3).IsNil() {
		init = getInt(v, 3, "match")
	}

	mStart, mEnd, caps, found := luaMatchFrom(s, pattern, int(init))
	if !found {
		v.Set(0, vm.Nil)
		return 1
	}

	if len(caps) == 0 {
		// No explicit captures, return whole match
		v.Set(0, vm.NewString(s[mStart:mEnd]))
		return 1
	}

	// Return all captures
	for i, c := range caps {
		if c.isPos {
			v.Set(i, vm.NewInt(int64(c.pos)))
		} else {
			v.Set(i, vm.NewString(c.str))
		}
	}
	return len(caps)
}

// string.gmatch(s, pattern [, init])
func stringGmatch(v *vm.VM) int {
	s := getString(v, 1, "gmatch")
	pattern := getString(v, 2, "gmatch")
	init := 1
	if !v.Get(3).IsNil() {
		init = int(getInt(v, 3, "gmatch"))
	}

	// Handle anchor
	searchPat := pattern
	if len(searchPat) > 0 && searchPat[0] == '^' {
		searchPat = searchPat[1:]
	}

	// Resolve negative init
	if init < 0 {
		init = len(s) + init + 1
	}
	if init < 1 {
		init = 1
	}

	pos := init - 1     // 0-based
	lastMatch := -1     // 0-based end of last match, -1 = none

	iter := vm.NewNativeFunc(func(v *vm.VM) int {
		for pos <= len(s) {
			end, caps, ok := luaMatchAt(s, searchPat, pos)
			if ok && end != lastMatch {
				// Valid match
				matchStart := pos
				lastMatch = end

				// Advance for next iteration
				if end == pos {
					pos++ // empty match: move forward 1
				} else {
					pos = end
				}

				// Return captures or whole match
				if len(caps) > 0 {
					for i, c := range caps {
						if c.isPos {
							v.Set(i, vm.NewInt(int64(c.pos)))
						} else {
							v.Set(i, vm.NewString(c.str))
						}
					}
					return len(caps)
				}
				v.Set(0, vm.NewString(s[matchStart:end]))
				return 1
			}
			pos++
		}
		v.Set(0, vm.Nil)
		return 1
	})

	v.Set(0, iter)
	return 1
}

// Helper functions

func getString(v *vm.VM, idx int, fname string) string {
	val := v.Get(idx)
	if val.IsString() {
		return val.AsString()
	}
	if val.IsNumber() {
		return val.String()
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (string expected, got %s)", idx, fname, val.Type()))
}

func getInt(v *vm.VM, idx int, fname string) int64 {
	val := v.Get(idx)
	if i, ok := val.ToInt(); ok {
		return i
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (number expected, got %s)", idx, fname, val.Type()))
}

func posRelat(pos int64, len int) int {
	if pos >= 0 {
		return int(pos)
	}
	return len + int(pos) + 1
}

func luaFormatValues(v *vm.VM, format string, vals []vm.Value) string {
	var result strings.Builder
	argIdx := 0

	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			result.WriteByte(format[i])
			continue
		}

		if i+1 >= len(format) {
			result.WriteByte('%')
			break
		}

		i++
		if format[i] == '%' {
			result.WriteByte('%')
			continue
		}

		// Parse flags and width/precision
		spec := "%"
		for i < len(format) && !strings.ContainsRune("diouxXeEfFgGaAcspq%", rune(format[i])) {
			if !strings.ContainsRune("#0- +.0123456789", rune(format[i])) {
				panic(fmt.Sprintf("invalid conversion '%%%c'", format[i]))
			}
			spec += string(format[i])
			i++
		}
		if i >= len(format) {
			panic(fmt.Sprintf("invalid conversion '%s'", spec))
		}

		// Validate width and precision (Lua 5.4: must be < 100)
		validateFormatWidthPrec(spec)

		specChar := format[i]

		if argIdx >= len(vals) {
			panic(fmt.Sprintf("bad argument #%d to 'format' (no value)", argIdx+2))
		}
		val := vals[argIdx]
		argIdx++

		switch specChar {
		case 'd', 'i':
			goSpec := spec + "d"
			if i, ok := val.ToInt(); ok {
				result.WriteString(fmt.Sprintf(goSpec, i))
			} else if _, ok := val.ToNumber(); ok {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number has no integer representation)", argIdx+1))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'u':
			goSpec := spec + "d"
			if i, ok := val.ToInt(); ok {
				result.WriteString(fmt.Sprintf(goSpec, uint64(i)))
			} else if _, ok := val.ToNumber(); ok {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number has no integer representation)", argIdx+1))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'o', 'x', 'X':
			goSpec := spec + string(specChar)
			if i, ok := val.ToInt(); ok {
				result.WriteString(fmt.Sprintf(goSpec, uint64(i)))
			} else if _, ok := val.ToNumber(); ok {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number has no integer representation)", argIdx+1))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'e', 'E', 'f', 'g', 'G':
			goSpec := spec
			// C default precision for %g/%G is 6; Go uses shortest-unique
			if (specChar == 'g' || specChar == 'G') && !strings.Contains(spec, ".") {
				goSpec = goSpec + ".6"
			}
			goSpec = goSpec + string(specChar)
			if n, ok := val.ToNumber(); ok {
				if special, ok := formatSpecialFloat(spec, specChar, n); ok {
					result.WriteString(special)
				} else {
					result.WriteString(fmt.Sprintf(goSpec, n))
				}
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'a', 'A':
			// Go's fmt package does not support %a/%A for floats.
			if n, ok := val.ToNumber(); ok {
				if special, ok := formatSpecialFloat(spec, specChar, n); ok {
					result.WriteString(special)
				} else {
					prec := -1 // default: shortest
					if dotIdx := strings.IndexByte(spec, '.'); dotIdx >= 0 {
						if p, err := strconv.Atoi(spec[dotIdx+1:]); err == nil {
							prec = p
						}
					}
					// Strip precision from spec for width/flags only
					widthSpec := spec
					if dotIdx := strings.IndexByte(widthSpec, '.'); dotIdx >= 0 {
						widthSpec = widthSpec[:dotIdx]
					}
					s := formatHexFloat(n, prec)
					if specChar == 'A' {
						s = strings.ToUpper(s)
					}
					if widthSpec != "%" {
						result.WriteString(fmt.Sprintf(widthSpec+"s", s))
					} else {
						result.WriteString(s)
					}
				}
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'F':
			panic("invalid conversion '%F' to 'format'")
		case 's':
			goSpec := spec + "s"
			result.WriteString(fmt.Sprintf(goSpec, tolstring(v, val)))
		case 'q':
			result.WriteString(luaQuote(val, argIdx+1))
		case 'c':
			if i, ok := val.ToInt(); ok {
				// Lua %c writes one byte (C unsigned char semantics).
				result.WriteByte(byte(i))
			} else if _, ok := val.ToNumber(); ok {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number has no integer representation)", argIdx+1))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'p':
			result.WriteString(luaPointerFormat(val))
		default:
			panic(fmt.Sprintf("invalid conversion '%%%c' to 'format'", specChar))
		}
	}

	return result.String()
}

// normalizeHexExponent rewrites strconv hex-float exponents (+00, -04, ...)
// to Lua-style minimal exponents (+0, -4, ...).
func normalizeHexExponent(s string) string {
	p := strings.LastIndexAny(s, "pP")
	if p == -1 || p+1 >= len(s) {
		return s
	}
	signIdx := p + 1
	sign := ""
	if s[signIdx] == '+' || s[signIdx] == '-' {
		sign = s[signIdx : signIdx+1]
		signIdx++
	}
	if signIdx >= len(s) {
		return s
	}
	digits := s[signIdx:]
	trimmed := strings.TrimLeft(digits, "0")
	if trimmed == "" {
		trimmed = "0"
	}
	return s[:p+1] + sign + trimmed
}

// formatHexFloat formats a float64 as a C-compatible hex float with the given
// precision (number of hex digits after the decimal point). Unlike Go's
// strconv.FormatFloat which renormalizes on carry, this preserves C-style
// output where the leading digit can be > 1 after rounding.
func formatHexFloat(f float64, prec int) string {
	// Get the full-precision representation from Go
	full := strconv.FormatFloat(f, 'x', -1, 64)
	full = normalizeHexExponent(full)

	if prec < 0 {
		return full
	}

	// Parse components
	neg := false
	s := full
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	s = s[2:] // skip "0x"

	pIdx := strings.IndexByte(s, 'p')
	mantStr := s[:pIdx]
	expStr := s[pIdx+1:]
	exp, _ := strconv.Atoi(expStr)

	// Parse leading digit and hex fraction digits
	lead := hexCharToInt(mantStr[0])
	var digits []int
	if dotIdx := strings.IndexByte(mantStr, '.'); dotIdx >= 0 {
		for i := dotIdx + 1; i < len(mantStr); i++ {
			digits = append(digits, hexCharToInt(mantStr[i]))
		}
	}

	// Pad to at least prec+1 digits for rounding
	for len(digits) <= prec {
		digits = append(digits, 0)
	}

	// Round at position prec
	if prec < len(digits) {
		roundDigit := digits[prec]
		digits = digits[:prec]
		if roundDigit >= 8 { // >= 0.5 in hex
			carry := 1
			for i := len(digits) - 1; i >= 0 && carry > 0; i-- {
				digits[i] += carry
				if digits[i] > 15 {
					digits[i] = 0
				} else {
					carry = 0
				}
			}
			if carry > 0 {
				lead += carry
				// Don't renormalize — C-style keeps the larger lead digit
			}
		}
	}

	// Build output
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteString("0x")
	b.WriteByte(intToHexChar(lead))
	if prec > 0 {
		b.WriteByte('.')
		for i := 0; i < prec; i++ {
			if i < len(digits) {
				b.WriteByte(intToHexChar(digits[i]))
			} else {
				b.WriteByte('0')
			}
		}
	}
	fmt.Fprintf(&b, "p%+d", exp)
	return b.String()
}

func hexCharToInt(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

func intToHexChar(v int) byte {
	if v < 10 {
		return byte('0' + v)
	}
	return byte('a' + v - 10)
}

func formatSpecialFloat(spec string, specChar byte, n float64) (string, bool) {
	if !math.IsInf(n, 0) && !math.IsNaN(n) {
		return "", false
	}

	upper := specChar == 'E' || specChar == 'G' || specChar == 'A'
	var token string
	if math.IsNaN(n) {
		if upper {
			token = "-NAN"
		} else {
			token = "-nan"
		}
	} else if math.IsInf(n, -1) {
		if upper {
			token = "-INF"
		} else {
			token = "-inf"
		}
	} else {
		if upper {
			token = "INF"
		} else {
			token = "inf"
		}
		if strings.Contains(spec, "+") {
			token = "+" + token
		}
	}

	width, left := parseFormatWidth(spec)
	if width > len(token) {
		pad := strings.Repeat(" ", width-len(token))
		if left {
			token += pad
		} else {
			token = pad + token
		}
	}
	return token, true
}

func parseFormatWidth(spec string) (width int, left bool) {
	i := 1 // skip '%'
	for i < len(spec) && strings.ContainsRune("#0- +", rune(spec[i])) {
		if spec[i] == '-' {
			left = true
		}
		i++
	}
	start := i
	for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
		i++
	}
	if i > start {
		if w, err := strconv.Atoi(spec[start:i]); err == nil {
			width = w
		}
	}
	return width, left
}

// validateFormatWidthPrec panics if width or precision >= 100 (Lua 5.4 limit).
func validateFormatWidthPrec(spec string) {
	i := 1 // skip '%'
	// skip flags
	for i < len(spec) && strings.ContainsRune("#0- +", rune(spec[i])) {
		i++
	}
	// parse width
	start := i
	for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
		i++
	}
	if i > start {
		if w, err := strconv.Atoi(spec[start:i]); err == nil && w >= 100 {
			panic("invalid format (width or precision too long)")
		}
	}
	// parse precision
	if i < len(spec) && spec[i] == '.' {
		i++
		start = i
		for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
			i++
		}
		if i > start {
			if p, err := strconv.Atoi(spec[start:i]); err == nil && p >= 100 {
				panic("invalid format (width or precision too long)")
			}
		}
	}
}

// luaQuote implements Lua's %q format for proper Lua-parseable quoting.
func luaQuote(val vm.Value, argIdx int) string {
	if val.IsNil() {
		return "nil"
	}
	if val.IsBool() {
		if val.AsBool() {
			return "true"
		}
		return "false"
	}
	if val.IsFloat() {
		f := val.AsFloat()
		if math.IsInf(f, 1) {
			return "1e9999"
		}
		if math.IsInf(f, -1) {
			return "-1e9999"
		}
		if math.IsNaN(f) {
			return "(0/0)"
		}
		// Use hex float format for exact roundtrip (matches Lua 5.4)
		s := strconv.FormatFloat(f, 'x', -1, 64)
		return normalizeHexExponent(s)
	}
	if val.IsInt() {
		i := val.AsInt()
		if i == math.MinInt64 {
			return "0x8000000000000000"
		}
		return fmt.Sprintf("%d", i)
	}
	if !val.IsString() {
		panic(fmt.Sprintf("bad argument #%d to 'format' (value has no literal form)", argIdx))
	}
	// String quoting — matches Lua 5.4 addquoted (lstrlib.c)
	s := valueToString(val)
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"' || ch == '\\':
			b.WriteByte('\\')
			b.WriteByte(ch)
		case ch == '\n':
			// Lua 5.4: backslash + literal newline
			b.WriteString("\\\n")
		case ch < 0x20 || ch == 0x7f:
			// Control character: use decimal escape.
			// Use 3-digit form if next byte is a digit.
			if i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
				b.WriteString(fmt.Sprintf("\\%03d", ch))
			} else {
				b.WriteString(fmt.Sprintf("\\%d", ch))
			}
		default:
			b.WriteByte(ch)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// luaPointerFormat implements Lua's %p format.
func luaPointerFormat(val vm.Value) string {
	if val.IsTable() {
		return fmt.Sprintf("%p", val.AsTable())
	}
	return "(null)"
}

func toInt(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	case int:
		return int64(x)
	default:
		return 0
	}
}

func toUint(v interface{}) uint64 {
	return uint64(toInt(v))
}

func toFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case int:
		return float64(x)
	default:
		return 0
	}
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprintf("%v", v)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func utf8Len(s string) int {
	return utf8.RuneCountInString(s)
}

// string.dump(function [, strip])
func stringDump(v *vm.VM) int {
	val := v.Get(1)
	if !val.IsFunction() {
		panic(fmt.Sprintf("bad argument #1 to 'dump' (function expected, got %s)", val.Type()))
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

	// Line info
	d.writeInt(int64(p.LineDef))
	d.writeInt(int64(p.LastLine))

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
