package stdlib

import (
	"bytes"
	"fmt"
	"strings"

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
	// __index points to the string table itself
	str.SetString("__index", vm.NewTable(str))
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

// string.gsub(s, pattern, repl [, n])
func stringGsub(v *vm.VM) int {
	s := getString(v, 1, "gsub")
	pattern := getString(v, 2, "gsub")
	repl := v.Get(3)
	// Validate replacement type — Lua 5.4 also accepts numbers (coerced to string)
	if repl.IsNumber() {
		repl = vm.NewString(valueToString(repl))
	} else if !repl.IsString() && !repl.IsFunction() && !repl.IsNativeFunc() && !repl.IsTable() {
		panic(fmt.Sprintf("bad argument #3 to 'gsub' (string/function/table expected, got %s)", repl.Type()))
	}
	maxRepl := -1
	if v.ArgCount() >= 4 && !v.Get(4).IsNil() {
		n := int(getInt(v, 4, "gsub"))
		if n < 0 {
			n = 0
		}
		maxRepl = n
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
				replacement = lookupGsubTable(v, repl, matchCaps, s[pos:end])
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
		if repl[i] == '%' {
			if i+1 >= len(repl) {
				panic("invalid use of '%' in replacement string")
			}
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
					panic(fmt.Sprintf("invalid capture index %%%c", next))
				}
				i++
				continue
			} else if next == '%' {
				result.WriteByte('%')
				i++
				continue
			}
			panic("invalid use of '%' in replacement string")
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
		panic(err)
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
func lookupGsubTable(v *vm.VM, repl vm.Value, captures []captureValue, wholeMatch string) string {
	var key vm.Value
	c := captures[0]
	if c.isPos {
		key = vm.NewInt(int64(c.pos))
	} else {
		key = vm.NewString(c.str)
	}
	val, err := v.TableGet(repl.AsTable(), key)
	if err != nil {
		panic(err.Error())
	}
	if val.IsString() {
		return val.AsString()
	} else if val.IsNumber() {
		return valueToString(val)
	} else if val.IsNil() || (val.IsBool() && !val.AsBool()) {
		return wholeMatch
	}
	panic(fmt.Sprintf("invalid replacement value (a %s)", val.Type()))
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


