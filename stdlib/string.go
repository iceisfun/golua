package stdlib

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/iceisfun/golua/v2/vm"
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
	str.SetString("pack", vm.NewNativeFunc(stringPack))
	str.SetString("unpack", vm.NewNativeFunc(stringUnpack))
	str.SetString("packsize", vm.NewNativeFunc(stringPacksize))

	strLib := vm.NewTable(str)
	v.SetGlobal("string", strLib)

	// Create a separate string metatable (distinct from the string library).
	// Lua 5.4: getmetatable("") ~= string (they are different tables).
	// The metatable has __index pointing to the string library and default
	// arithmetic metamethods that perform string-to-number coercion.
	strMeta := vm.NewEmptyTable()
	strMeta.SetString("__index", strLib)

	// Lua 5.4 default string arithmetic metamethods: coerce to number.
	// If user sets mt.__add = nil, coercion stops (matches Lua 5.4).
	strMeta.SetString("__add", vm.NewNativeFunc(makeStringArith("add", func(a, b float64) float64 { return a + b }, func(a, b int64) int64 { return a + b })))
	strMeta.SetString("__sub", vm.NewNativeFunc(makeStringArith("sub", func(a, b float64) float64 { return a - b }, func(a, b int64) int64 { return a - b })))
	strMeta.SetString("__mul", vm.NewNativeFunc(makeStringArith("mul", func(a, b float64) float64 { return a * b }, func(a, b int64) int64 { return a * b })))
	strMeta.SetString("__div", vm.NewNativeFunc(makeStringArith("div", func(a, b float64) float64 { return a / b }, nil)))
	strMeta.SetString("__idiv", vm.NewNativeFunc(makeStringArithFloorDiv()))
	strMeta.SetString("__mod", vm.NewNativeFunc(makeStringArithMod()))
	strMeta.SetString("__pow", vm.NewNativeFunc(makeStringArith("pow", func(a, b float64) float64 { return math.Pow(a, b) }, nil)))
	strMeta.SetString("__unm", vm.NewNativeFunc(stringMetaUnm))

	v.SetStringMeta(strMeta)
}

// string.len(s)
func stringLen(v *vm.VM) int {
	s := getString(v, 1, "string.len")
	v.Set(0, vm.NewInt(int64(len(s))))
	return 1
}

// string.sub(s, i [, j])
func stringSub(v *vm.VM) int {
	s := getString(v, 1, "string.sub")
	i := getInt(v, 2, "string.sub")
	j := int64(len(s))
	if !v.Get(3).IsNil() {
		j = getInt(v, 3, "string.sub")
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
	s := getString(v, 1, "string.upper")
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
	s := getString(v, 1, "string.lower")
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
	s := getString(v, 1, "string.rep")
	n := getInt(v, 2, "string.rep")
	sep := ""
	if !v.Get(3).IsNil() {
		sep = getString(v, 3, "string.rep")
	}

	if n <= 0 {
		v.Set(0, vm.NewString(""))
		return 1
	}

	// Check for overflow: total = len(s)*n + len(sep)*(n-1)
	// Use a limit well below Go's allocation ceiling to prevent unrecoverable
	// runtime OOM panics that bypass pcall/recover.
	const maxSize int64 = 1<<30 - 1 // ~1GB, must reject before Go allocator fails
	sLen := int64(len(s))
	sepLen := int64(len(sep))
	totalSize := sLen*n + sepLen*(n-1)
	if totalSize < 0 || totalSize > maxSize || (sLen > 0 && n > maxSize/sLen) {
		panic("resulting string too large")
	}

	var result strings.Builder
	result.Grow(int(totalSize))
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
	s := getString(v, 1, "string.reverse")
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	v.Set(0, vm.NewString(string(b)))
	return 1
}

// string.byte(s [, i [, j]])
func stringByte(v *vm.VM) int {
	s := getString(v, 1, "string.byte")
	i := int64(1)
	if !v.Get(2).IsNil() {
		i = getInt(v, 2, "string.byte")
	}
	j := i
	if !v.Get(3).IsNil() {
		j = getInt(v, 3, "string.byte")
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
		c := getInt(v, i, "string.char")
		if c < 0 || c > 255 {
			callerArgError(v, i, "string.char", "value out of range")
		}
		buf.WriteByte(byte(c))
	}
	v.Set(0, vm.NewString(buf.String()))
	return 1
}

// hasPatternSpecials returns true if the pattern contains any Lua pattern
// special characters. Matches Lua 5.4's SPECIALS = "^$*+?.([%-".
// Note: ')' and ']' are intentionally NOT in this set.
func hasPatternSpecials(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '^', '$', '*', '+', '?', '.', '(', '[', '%', '-':
			return true
		}
	}
	return false
}

// string.find(s, pattern [, init [, plain]])
func stringFind(v *vm.VM) int {
	s := getString(v, 1, "string.find")
	pattern := getString(v, 2, "string.find")
	init := int64(1)
	if !v.Get(3).IsNil() {
		init = getInt(v, 3, "string.find")
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

	// Lua 5.4 optimization: if pattern contains no special characters,
	// use plain substring search. The specials set matches Lua 5.4's
	// SPECIALS string "^$*+?.([%-" — note that ')' and ']' are NOT
	// considered specials, so patterns like "a)b" do a plain search.
	if !plain && !hasPatternSpecials(pattern) {
		plain = true
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
	checkCaptures(caps)

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
	s := getString(v, 1, "string.gsub")
	pattern := getString(v, 2, "string.gsub")
	repl := v.Get(3)
	// Validate replacement type — Lua 5.4 also accepts numbers (coerced to string)
	if repl.IsNumber() {
		repl = vm.NewString(valueToString(repl))
	} else if !repl.IsString() && !repl.IsFunction() && !repl.IsNativeFunc() && !repl.IsTable() {
		got := repl.Type()
		if v.ArgCount() < 3 {
			got = "no value"
		}
		callerArgError(v, 3, "string.gsub", fmt.Sprintf("string/function/table expected, got %s", got))
	}
	maxRepl := -1
	if v.ArgCount() >= 4 && !v.Get(4).IsNil() {
		n := int(getInt(v, 4, "string.gsub"))
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
	changed := false   // track whether any substitution modified text
	pos := 0           // 0-based current position
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
			substituted := false // true if repl function/table produced a value
			if repl.IsString() {
				// For string replacements, unfinished captures are only
				// an error if %N actually references them.
				replacement = expandReplacement(repl.AsString(), s, pos, end, matchCaps)
				substituted = replacement != s[pos:end]
			} else if repl.IsFunction() || repl.IsNativeFunc() {
				checkCaptures(matchCaps)
				replacement, substituted = callGsubFunc(v, repl, matchCaps, s[pos:end])
			} else if repl.IsTable() {
				checkCaptures(matchCaps)
				replacement, substituted = lookupGsubTable(v, repl, matchCaps, s[pos:end])
			}

			if substituted {
				changed = true
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

	// Optimization: when no substitution changed any text, return the
	// original string object (same pointer identity, matching Lua 5.4).
	if !changed {
		v.Set(0, v.Get(1))
	} else {
		v.Set(0, vm.NewString(result.String()))
	}
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
					if c.unfinished {
						panic("unfinished capture")
					}
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
// Returns the replacement string and whether a substitution occurred
// (true when the function returned a non-nil, non-false value).
func callGsubFunc(v *vm.VM, fn vm.Value, captures []captureValue, wholeMatch string) (string, bool) {
	args := make([]vm.Value, len(captures))
	for i, cap := range captures {
		if cap.isPos {
			args[i] = vm.NewInt(int64(cap.pos))
		} else {
			args[i] = vm.NewString(cap.str)
		}
	}

	// Use ProtectedCall but re-panic on error (Lua propagates gsub function errors)
	exitNonYieldable := v.EnterNonYieldable()
	defer exitNonYieldable()
	results, err := v.ProtectedCall(fn, args)
	if err != nil {
		panic(err)
	}
	if len(results) == 0 {
		return wholeMatch, false
	}

	ret := results[0]
	if ret.IsString() {
		return ret.AsString(), true
	} else if ret.IsNumber() {
		return valueToString(ret), true
	} else if ret.IsNil() || (ret.IsBool() && !ret.AsBool()) {
		return wholeMatch, false
	}
	panic(fmt.Sprintf("invalid replacement value (a %s)", ret.Type()))
}

// lookupGsubTable looks up a gsub replacement from a table.
// Returns the replacement string and whether a substitution occurred.
func lookupGsubTable(v *vm.VM, repl vm.Value, captures []captureValue, wholeMatch string) (string, bool) {
	var key vm.Value
	c := captures[0]
	if c.isPos {
		key = vm.NewInt(int64(c.pos))
	} else {
		key = vm.NewString(c.str)
	}
	val, err := v.TableGet(repl.AsTable(), key)
	if err != nil {
		panic(err)
	}
	if val.IsString() {
		return val.AsString(), true
	} else if val.IsNumber() {
		return valueToString(val), true
	} else if val.IsNil() || (val.IsBool() && !val.AsBool()) {
		return wholeMatch, false
	}
	panic(fmt.Sprintf("invalid replacement value (a %s)", val.Type()))
}

// string.match(s, pattern [, init])
func stringMatch(v *vm.VM) int {
	s := getString(v, 1, "string.match")
	pattern := getString(v, 2, "string.match")
	init := int64(1)
	if !v.Get(3).IsNil() {
		init = getInt(v, 3, "string.match")
	}

	mStart, mEnd, caps, found := luaMatchFrom(s, pattern, int(init))
	if !found {
		v.Set(0, vm.Nil)
		return 1
	}
	checkCaptures(caps)

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
	s := getString(v, 1, "string.gmatch")
	pattern := getString(v, 2, "string.gmatch")
	init := 1
	if !v.Get(3).IsNil() {
		init = int(getInt(v, 3, "string.gmatch"))
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

	iter := vm.NewNativeFuncWithNups(func(v *vm.VM) int {
		for pos <= len(s) {
			end, caps, ok := luaMatchAt(s, pattern, pos)
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
					checkCaptures(caps)
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
		return 0
	}, 3)

	// Store upvalues for debug.getupvalue introspection (matches Lua 5.4 C closure)
	iter.SetNativeFuncUpvalue(1, vm.NewString(s))
	iter.SetNativeFuncUpvalue(2, vm.NewString(pattern))
	// Upvalue 3 is internal position state (userdata in Lua 5.4)

	v.Set(0, iter)
	return 1
}

// makeStringArith creates a string arithmetic metamethod that coerces
// operands to numbers and performs the operation. If one operand cannot
// be coerced, it falls back to the other operand's metamethod (matching
// Lua 5.4.6+ behavior).
func makeStringArith(opName string, floatOp func(float64, float64) float64, intOp func(int64, int64) int64) func(*vm.VM) int {
	mmName := "__" + opName
	return func(v *vm.VM) int {
		a, b := v.Get(1), v.Get(2)
		cv1, ok1 := coerceToNumber(a)
		cv2, ok2 := coerceToNumber(b)
		if !ok1 || !ok2 {
			// Try the other operand's metamethod (skip strings to avoid recursion)
			if result, ok := stringArithFallback(v, a, b, ok1, ok2, mmName); ok {
				v.Set(0, result)
				return 1
			}
			panic(fmt.Sprintf("attempt to %s a '%s' with a '%s'", opName, a.Type(), b.Type()))
		}
		// Integer path (when both are int and intOp is available)
		if intOp != nil && cv1.IsInt() && cv2.IsInt() {
			v.Set(0, vm.NewInt(intOp(cv1.AsInt(), cv2.AsInt())))
			return 1
		}
		n1, _ := cv1.ToNumber()
		n2, _ := cv2.ToNumber()
		v.Set(0, vm.NewFloat(floatOp(n1, n2)))
		return 1
	}
}

func makeStringArithFloorDiv() func(*vm.VM) int {
	return func(v *vm.VM) int {
		a, b := v.Get(1), v.Get(2)
		cv1, ok1 := coerceToNumber(a)
		cv2, ok2 := coerceToNumber(b)
		if !ok1 || !ok2 {
			if result, ok := stringArithFallback(v, a, b, ok1, ok2, "__idiv"); ok {
				v.Set(0, result)
				return 1
			}
			panic(fmt.Sprintf("attempt to idiv a '%s' with a '%s'", a.Type(), b.Type()))
		}
		if cv1.IsInt() && cv2.IsInt() {
			i1, i2 := cv1.AsInt(), cv2.AsInt()
			if i2 == 0 {
				panic("attempt to divide by zero")
			}
			if i2 == -1 {
				v.Set(0, vm.NewInt(-i1))
				return 1
			}
			q := i1 / i2
			if (i1^i2) < 0 && q*i2 != i1 {
				q--
			}
			v.Set(0, vm.NewInt(q))
			return 1
		}
		n1, _ := cv1.ToNumber()
		n2, _ := cv2.ToNumber()
		v.Set(0, vm.NewFloat(math.Floor(n1/n2)))
		return 1
	}
}

func makeStringArithMod() func(*vm.VM) int {
	return func(v *vm.VM) int {
		a, b := v.Get(1), v.Get(2)
		cv1, ok1 := coerceToNumber(a)
		cv2, ok2 := coerceToNumber(b)
		if !ok1 || !ok2 {
			if result, ok := stringArithFallback(v, a, b, ok1, ok2, "__mod"); ok {
				v.Set(0, result)
				return 1
			}
			panic(fmt.Sprintf("attempt to mod a '%s' with a '%s'", a.Type(), b.Type()))
		}
		if cv1.IsInt() && cv2.IsInt() {
			i1, i2 := cv1.AsInt(), cv2.AsInt()
			if i2 == 0 {
				panic("attempt to perform 'n%0'")
			}
			if i2 == -1 {
				v.Set(0, vm.NewInt(0))
				return 1
			}
			r := i1 % i2
			if r != 0 && (r^i2) < 0 {
				r += i2
			}
			v.Set(0, vm.NewInt(r))
			return 1
		}
		n1, _ := cv1.ToNumber()
		n2, _ := cv2.ToNumber()
		result := math.Mod(n1, n2)
		if result != 0 && (result < 0) != (n2 < 0) {
			result += n2
		}
		v.Set(0, vm.NewFloat(result))
		return 1
	}
}

// stringMetaUnm is the default string unary minus metamethod.
func stringMetaUnm(v *vm.VM) int {
	a := v.Get(1)
	nv, ok := vm.StringToNumericValue(a.AsString())
	if !ok {
		panic(fmt.Sprintf("attempt to unm a '%s' with a '%s'", a.Type(), a.Type()))
	}
	if nv.IsInt() {
		v.Set(0, vm.NewInt(-nv.AsInt()))
	} else {
		v.Set(0, vm.NewFloat(-nv.AsFloat()))
	}
	return 1
}

// stringArithFallback tries the other operand's metamethod when the string
// metamethod can't coerce one of the operands. Returns (result, true) if a
// fallback metamethod was found and called successfully.
func stringArithFallback(v *vm.VM, a, b vm.Value, ok1, ok2 bool, mmName string) (vm.Value, bool) {
	// Try each non-string operand's metamethod. When both coercions fail,
	// we need to check both sides (e.g. "hello" + table_with___add).
	for _, other := range [2]vm.Value{a, b} {
		// Skip strings (would recurse back to us)
		if other.IsString() {
			continue
		}
		mm := v.GetMetafield(other, mmName)
		if !mm.IsNil() {
			results, err := v.ProtectedCall(mm, []vm.Value{a, b})
			if err != nil {
				panic(err.Error())
			}
			if len(results) > 0 {
				return results[0], true
			}
			return vm.Nil, true
		}
	}
	return vm.Nil, false
}

func coerceToNumber(val vm.Value) (vm.Value, bool) {
	if val.IsInt() || val.IsFloat() {
		return val, true
	}
	if val.IsString() {
		nv, ok := vm.StringToNumericValue(val.AsString())
		return nv, ok
	}
	return vm.Nil, false
}

func pickNonNumericType(a, b string) string {
	if a == "string" {
		return a
	}
	return b
}
