package stdlib

import (
	"bytes"
	"fmt"
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

// string.upper(s)
func stringUpper(v *vm.VM) int {
	s := getString(v, 1, "upper")
	v.Set(0, vm.NewString(strings.ToUpper(s)))
	return 1
}

// string.lower(s)
func stringLower(v *vm.VM) int {
	s := getString(v, 1, "lower")
	v.Set(0, vm.NewString(strings.ToLower(s)))
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
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	v.Set(0, vm.NewString(string(runes)))
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

	searchStr := s[start-1:]

	if plain {
		// Plain string search
		idx := strings.Index(searchStr, pattern)
		if idx == -1 {
			v.Set(0, vm.Nil)
			return 1
		}
		v.Set(0, vm.NewInt(int64(start+idx)))
		v.Set(1, vm.NewInt(int64(start+idx+len(pattern)-1)))
		return 2
	}

	// Pattern matching via Lua pattern engine
	matchInfo := luaMatchWithPos(s, pattern, start)
	if matchInfo == nil {
		v.Set(0, vm.Nil)
		return 1
	}

	// Always return 1-based start and end (inclusive)
	v.Set(0, vm.NewInt(int64(matchInfo.start+1)))
	v.Set(1, vm.NewInt(int64(matchInfo.end)))
	nret := 2

	// If the pattern had explicit captures, return them after positions
	if matchInfo.hasExplicitCaptures {
		for i, cap := range matchInfo.captures {
			v.Set(2+i, vm.NewString(cap))
		}
		nret += len(matchInfo.captures)
	}

	return nret
}

// string.format(formatstring, ...)
func stringFormat(v *vm.VM) int {
	format := getString(v, 1, "format")
	args := make([]interface{}, v.ArgCount()-1)
	for i := 2; i <= v.ArgCount(); i++ {
		val := v.Get(i)
		switch {
		case val.IsInt():
			args[i-2] = val.AsInt()
		case val.IsFloat():
			args[i-2] = val.AsFloat()
		case val.IsString():
			args[i-2] = val.AsString()
		case val.IsBool():
			args[i-2] = val.AsBool()
		default:
			args[i-2] = valueToString(val)
		}
	}

	// Convert Lua format to Go format
	result := luaFormatToGo(format, args)
	v.Set(0, vm.NewString(result))
	return 1
}

// string.gsub(s, pattern, repl [, n])
func stringGsub(v *vm.VM) int {
	s := getString(v, 1, "gsub")
	pattern := getString(v, 2, "gsub")
	repl := v.Get(3)
	maxRepl := -1 // replace all
	if v.ArgCount() >= 4 && !v.Get(4).IsNil() {
		maxRepl = int(getInt(v, 4, "gsub"))
	}

	var result strings.Builder
	count := 0
	pos := 0

	for pos <= len(s) && (maxRepl < 0 || count < maxRepl) {
		// Find next match
		matchInfo := luaMatchWithPos(s, pattern, pos+1) // 1-based init
		if matchInfo == nil {
			break
		}

		// Append text before the match
		result.WriteString(s[pos:matchInfo.start])

		// Get replacement
		var replacement string
		if repl.IsString() {
			replacement = expandReplacement(repl.AsString(), s, matchInfo)
		} else if repl.IsFunction() || repl.IsNativeFunc() {
			// Call function with captures (or whole match if no captures)
			replacement = callGsubFunc(v, repl, matchInfo.captures)
		} else if repl.IsTable() {
			// Table lookup with first capture (or whole match)
			key := matchInfo.captures[0]
			val := repl.AsTable().Get(vm.NewString(key))
			if val.IsString() {
				replacement = val.AsString()
			} else if val.IsNil() || (val.IsBool() && !val.AsBool()) {
				replacement = matchInfo.captures[0] // keep original
			} else {
				replacement = valueToString(val)
			}
		}

		result.WriteString(replacement)
		count++

		// Move past the match (but at least 1 char to avoid infinite loop on empty matches)
		if matchInfo.end > pos {
			pos = matchInfo.end
		} else {
			// Empty match: copy the character at current position before advancing
			if pos < len(s) {
				result.WriteByte(s[pos])
			}
			pos++
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

// matchWithPos holds match result with position info
type matchWithPos struct {
	start               int      // 0-based start position in string
	end                 int      // 0-based end position (exclusive)
	captures            []string // captured groups (or whole match if no groups)
	hasExplicitCaptures bool     // true when pattern had () groups
}

// luaMatchWithPos returns match info including positions
func luaMatchWithPos(s, pattern string, init int) *matchWithPos {
	if init < 0 {
		init = len(s) + init + 1
	}
	if init < 1 {
		init = 1
	}
	if init > len(s)+1 {
		return nil
	}

	// Handle anchored patterns
	anchored := false
	searchPat := pattern
	if len(searchPat) > 0 && searchPat[0] == '^' {
		anchored = true
		searchPat = searchPat[1:]
	}

	start := init - 1 // convert to 0-based
	if anchored {
		caps := matchPattern(s, start, searchPat, 0)
		if caps != nil {
			return &matchWithPos{
				start:               caps.start,
				end:                 caps.end,
				captures:            caps.captures(s),
				hasExplicitCaptures: len(caps.caps) > 0,
			}
		}
		return nil
	}

	for i := start; i <= len(s); i++ {
		caps := matchPattern(s, i, searchPat, 0)
		if caps != nil {
			return &matchWithPos{
				start:               i,
				end:                 caps.end,
				captures:            caps.captures(s),
				hasExplicitCaptures: len(caps.caps) > 0,
			}
		}
	}
	return nil
}

// expandReplacement expands a replacement string with captures
func expandReplacement(repl string, s string, m *matchWithPos) string {
	var result strings.Builder
	for i := 0; i < len(repl); i++ {
		if repl[i] == '%' && i+1 < len(repl) {
			next := repl[i+1]
			if next >= '0' && next <= '9' {
				idx := int(next - '0')
				if idx == 0 {
					// %0 = whole match
					result.WriteString(s[m.start:m.end])
				} else if idx <= len(m.captures) {
					result.WriteString(m.captures[idx-1])
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

// callGsubFunc calls a function for gsub replacement
func callGsubFunc(v *vm.VM, fn vm.Value, captures []string) string {
	// Build args
	args := make([]vm.Value, len(captures))
	for i, cap := range captures {
		args[i] = vm.NewString(cap)
	}

	// Use ProtectedCall
	results, err := v.ProtectedCall(fn, args)
	if err != nil || len(results) == 0 {
		return captures[0] // keep original on error
	}

	ret := results[0]
	if ret.IsString() {
		return ret.AsString()
	} else if ret.IsNil() || (ret.IsBool() && !ret.AsBool()) {
		return captures[0] // keep original if nil/false returned
	}
	return valueToString(ret)
}

// string.match(s, pattern [, init])
func stringMatch(v *vm.VM) int {
	s := getString(v, 1, "match")
	pattern := getString(v, 2, "match")
	init := int64(1)
	if !v.Get(3).IsNil() {
		init = getInt(v, 3, "match")
	}

	matches := luaMatch(s, pattern, int(init))
	if matches == nil {
		v.Set(0, vm.Nil)
		return 1
	}

	// Return all captures
	for i, m := range matches {
		v.Set(i, vm.NewString(m))
	}
	return len(matches)
}

// string.gmatch(s, pattern)
func stringGmatch(v *vm.VM) int {
	s := getString(v, 1, "gmatch")
	pattern := getString(v, 2, "gmatch")

	// 1-based position for luaMatchWithPos
	pos := 1

	iter := vm.NewNativeFunc(func(v *vm.VM) int {
		if pos > len(s)+1 {
			v.Set(0, vm.Nil)
			return 1
		}

		matchInfo := luaMatchWithPos(s, pattern, pos)
		if matchInfo == nil {
			v.Set(0, vm.Nil)
			return 1
		}

		// Advance past match; at least 1 byte for empty matches
		if matchInfo.end > matchInfo.start {
			pos = matchInfo.end + 1 // 0-based exclusive → 1-based
		} else {
			pos = matchInfo.start + 2 // advance past current char (1-based)
		}

		// Return explicit captures if present, otherwise whole match
		if matchInfo.hasExplicitCaptures {
			for i, cap := range matchInfo.captures {
				v.Set(i, vm.NewString(cap))
			}
			return len(matchInfo.captures)
		}
		v.Set(0, vm.NewString(matchInfo.captures[0]))
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

func luaFormatToGo(format string, args []interface{}) string {
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

		// Parse format specifier
		spec := "%"
		for i < len(format) && !strings.ContainsRune("diouxXeEfFgGaAcspq", rune(format[i])) {
			spec += string(format[i])
			i++
		}
		if i < len(format) {
			specChar := format[i]
			spec += string(specChar)

			if argIdx < len(args) {
				arg := args[argIdx]
				argIdx++

				switch specChar {
				case 'd', 'i':
					result.WriteString(fmt.Sprintf(spec, toInt(arg)))
				case 'o', 'u', 'x', 'X':
					result.WriteString(fmt.Sprintf(spec, toUint(arg)))
				case 'e', 'E', 'f', 'F', 'g', 'G':
					result.WriteString(fmt.Sprintf(spec, toFloat(arg)))
				case 's':
					result.WriteString(fmt.Sprintf(spec, toString(arg)))
				case 'q':
					result.WriteString(fmt.Sprintf("%q", toString(arg)))
				case 'c':
					result.WriteString(string(rune(toInt(arg))))
				default:
					result.WriteString(fmt.Sprintf(spec, arg))
				}
			}
		}
	}

	return result.String()
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

// For UTF-8 support
func utf8Len(s string) int {
	return utf8.RuneCountInString(s)
}
