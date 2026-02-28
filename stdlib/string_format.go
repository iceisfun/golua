package stdlib

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/iceisfun/golua/vm"
)

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
