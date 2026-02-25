package stdlib

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

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

	result := luaFormatValues(format, vals)
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

func luaFormatValues(format string, vals []vm.Value) string {
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
		for i < len(format) && !strings.ContainsRune("diouxXeEfFgGaAcspq", rune(format[i])) {
			spec += string(format[i])
			i++
		}
		if i >= len(format) {
			result.WriteString(spec)
			break
		}

		specChar := format[i]

		if argIdx >= len(vals) {
			panic(fmt.Sprintf("bad argument #%d to 'format' (no value)", argIdx+2))
		}
		val := vals[argIdx]
		argIdx++

		switch specChar {
		case 'd', 'i':
			goSpec := spec + "d"
			if val.IsInt() {
				result.WriteString(fmt.Sprintf(goSpec, val.AsInt()))
			} else if n, ok := val.ToNumber(); ok {
				result.WriteString(fmt.Sprintf(goSpec, int64(n)))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'u':
			goSpec := spec + "d"
			if val.IsInt() {
				result.WriteString(fmt.Sprintf(goSpec, uint64(val.AsInt())))
			} else if n, ok := val.ToNumber(); ok {
				result.WriteString(fmt.Sprintf(goSpec, uint64(n)))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'o', 'x', 'X':
			goSpec := spec + string(specChar)
			if val.IsInt() {
				result.WriteString(fmt.Sprintf(goSpec, uint64(val.AsInt())))
			} else if n, ok := val.ToNumber(); ok {
				result.WriteString(fmt.Sprintf(goSpec, uint64(n)))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'e', 'E', 'f', 'F', 'g', 'G', 'a', 'A':
			goSpec := spec + string(specChar)
			if n, ok := val.ToNumber(); ok {
				result.WriteString(fmt.Sprintf(goSpec, n))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 's':
			goSpec := spec + "s"
			result.WriteString(fmt.Sprintf(goSpec, valueToString(val)))
		case 'q':
			result.WriteString(luaQuote(val))
		case 'c':
			if val.IsInt() {
				result.WriteString(string(rune(val.AsInt())))
			} else if n, ok := val.ToNumber(); ok {
				result.WriteString(string(rune(int64(n))))
			} else {
				panic(fmt.Sprintf("bad argument #%d to 'format' (number expected, got %s)", argIdx+1, val.Type()))
			}
		case 'p':
			result.WriteString(luaPointerFormat(val))
		default:
			result.WriteString(spec + string(specChar))
		}
	}

	return result.String()
}

// luaQuote implements Lua's %q format for proper Lua-parseable quoting.
func luaQuote(val vm.Value) string {
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
		return fmt.Sprintf("%.17g", f)
	}
	if val.IsInt() {
		return fmt.Sprintf("%d", val.AsInt())
	}
	// String quoting
	s := valueToString(val)
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '"':
			b.WriteString("\\\"")
		case 0:
			b.WriteString("\\0")
		case 0x1a: // Ctrl-Z
			b.WriteString("\\26")
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
