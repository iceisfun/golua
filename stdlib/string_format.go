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
	format := getString(v, 1, "string.format")
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
			// Trailing % with no conversion specifier.
			// Lua 5.4 checks for an argument first, then errors on the
			// missing conversion character.
			if argIdx >= len(vals) {
				callerArgError(v, argIdx+2, "string.format", "no value")
			}
			argIdx++
			panic("invalid conversion '%\x00' to 'format'")
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
				// Lua 5.4: check for argument existence before reporting invalid conversion
				if argIdx >= len(vals) {
					callerArgError(v, argIdx+2, "string.format", "no value")
				}
				panic(fmt.Sprintf("invalid conversion '%s%c' to 'format'", spec, format[i]))
			}
			spec += string(format[i])
			i++
		}
		if i >= len(format) {
			if argIdx >= len(vals) {
				callerArgError(v, argIdx+2, "string.format", "no value")
			}
			argIdx++
			panic(fmt.Sprintf("invalid conversion '%s' to 'format'", spec))
		}

		specChar := format[i]

		// Validate spec structure: %[flags][width][.precision]
		validateFormatStructure(spec, specChar)

		// Validate width and precision (Lua 5.4: must be < 100)
		validateFormatWidthPrec(spec, specChar)

		// Validate flag/conversion combinations (Lua 5.4 restrictions)
		validateConversion(spec, specChar)

		if argIdx >= len(vals) {
			callerArgError(v, argIdx+2, "string.format", "no value")
		}
		val := vals[argIdx]
		argIdx++

		switch specChar {
		case 'd', 'i':
			goSpec := spec + "d"
			if i, ok := val.ToInt(); ok {
				result.WriteString(fmt.Sprintf(goSpec, i))
			} else if _, ok := val.ToNumber(); ok {
				callerArgError(v, argIdx+1, "string.format", "number has no integer representation")
			} else {
				callerArgError(v, argIdx+1, "string.format", fmt.Sprintf("number expected, got %s", val.Type()))
			}
		case 'u':
			goSpec := spec + "d"
			if i, ok := val.ToInt(); ok {
				result.WriteString(fmt.Sprintf(goSpec, uint64(i)))
			} else if _, ok := val.ToNumber(); ok {
				callerArgError(v, argIdx+1, "string.format", "number has no integer representation")
			} else {
				callerArgError(v, argIdx+1, "string.format", fmt.Sprintf("number expected, got %s", val.Type()))
			}
		case 'o', 'x', 'X':
			if i, ok := val.ToInt(); ok {
				result.WriteString(formatIntHex(spec, specChar, uint64(i)))
			} else if _, ok := val.ToNumber(); ok {
				callerArgError(v, argIdx+1, "string.format", "number has no integer representation")
			} else {
				callerArgError(v, argIdx+1, "string.format", fmt.Sprintf("number expected, got %s", val.Type()))
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
				callerArgError(v, argIdx+1, "string.format", fmt.Sprintf("number expected, got %s", val.Type()))
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
					hashFlag := strings.Contains(spec, "#")
				s := formatHexFloat(n, prec, hashFlag)
					if specChar == 'A' {
						s = strings.ToUpper(s)
					}
					// Apply sign flags (+/space)
					if n >= 0 && !math.Signbit(n) {
						if strings.Contains(widthSpec, "+") {
							s = "+" + s
						} else if strings.Contains(widthSpec, " ") {
							s = " " + s
						}
					}
					// Apply width with proper zero-padding (after sign+prefix)
					isZeroPad, width, leftAlign, _ := parseFormatFlags(widthSpec)
					if width > 0 && len(s) < width {
						pad := width - len(s)
						if leftAlign {
							s = s + strings.Repeat(" ", pad)
						} else if isZeroPad != 0 {
							// Zero-pad after sign and "0x"/"0X" prefix
							s = hexFloatZeroPad(s, width)
						} else {
							s = strings.Repeat(" ", pad) + s
						}
					}
					result.WriteString(s)
				}
			} else {
				callerArgError(v, argIdx+1, "string.format", fmt.Sprintf("number expected, got %s", val.Type()))
			}
		case 'F':
			panic(fmt.Sprintf("invalid conversion '%sF' to 'format'", spec))
		case 's':
			str := tolstring(v, val)
			if spec != "%" && strings.ContainsRune(str, 0) {
				callerArgError(v, argIdx+1, "string.format", "string contains zeros")
			}
			if spec == "%" {
				// No modifiers: embed string directly (preserving null bytes)
				result.WriteString(str)
			} else {
				// Lua 5.4 counts bytes (not runes) for width/precision
				result.WriteString(formatStringByBytes(spec, str))
			}
		case 'q':
			result.WriteString(luaQuote(v, val, argIdx+1))
		case 'c':
			if i, ok := val.ToInt(); ok {
				// Lua %c writes one byte (C unsigned char semantics).
				ch := string([]byte{byte(i)})
				// Apply width formatting if specified (e.g., %-16c).
				if spec != "%" {
					goSpec := spec + "s"
					ch = fmt.Sprintf(goSpec, ch)
				}
				result.WriteString(ch)
			} else if _, ok := val.ToNumber(); ok {
				callerArgError(v, argIdx+1, "string.format", "number has no integer representation")
			} else {
				callerArgError(v, argIdx+1, "string.format", fmt.Sprintf("number expected, got %s", val.Type()))
			}
		case 'p':
			ps := luaPointerFormat(val)
			if spec != "%" {
				goSpec := spec + "s"
				ps = fmt.Sprintf(goSpec, ps)
			}
			result.WriteString(ps)
		default:
			panic(fmt.Sprintf("invalid conversion '%s%c' to 'format'", spec, specChar))
		}
	}

	return result.String()
}

// hexFloatZeroPad inserts zero padding after the sign + "0x"/"0X" prefix
// in a hex float string, matching C printf behavior for %0Na.
func hexFloatZeroPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	// Find the position after sign + "0x"/"0X"
	pos := 0
	if pos < len(s) && (s[pos] == '-' || s[pos] == '+') {
		pos++
	}
	if pos+1 < len(s) && s[pos] == '0' && (s[pos+1] == 'x' || s[pos+1] == 'X') {
		pos += 2
	}
	pad := strings.Repeat("0", width-len(s))
	return s[:pos] + pad + s[pos:]
}

// formatIntHex handles %o, %x, %X with proper C-compatible width calculation.
// Go's fmt.Sprintf with # + 0 flags doesn't count the "0x" prefix in the width,
// but C/Lua do. This function handles that case manually.
func formatIntHex(spec string, conv byte, val uint64) string {
	// Parse flags (before width digits)
	hasHash := false
	hasZero := false
	for j := 1; j < len(spec); j++ {
		c := spec[j]
		if c == '#' || c == '0' || c == '-' || c == ' ' || c == '+' {
			if c == '#' {
				hasHash = true
			}
			if c == '0' {
				hasZero = true
			}
		} else {
			break // width digits start
		}
	}

	// C printf: # flag has no effect on hex when value is 0 (no 0x prefix).
	// But for octal, # always ensures a leading 0 (even for value 0).
	if val == 0 && conv != 'o' {
		hasHash = false
	}

	// If we have both # and 0 flags with a width for x/X, handle manually
	// to ensure the 0x/0X prefix is counted in the width.
	if hasHash && hasZero && (conv == 'x' || conv == 'X') {
		width, leftAlign := parseFormatWidth(spec)
		if width > 0 && !leftAlign {
			// Format digits without prefix or padding
			var digits string
			if conv == 'X' {
				digits = fmt.Sprintf("%X", val)
			} else {
				digits = fmt.Sprintf("%x", val)
			}
			prefix := "0x"
			if conv == 'X' {
				prefix = "0X"
			}
			totalLen := len(prefix) + len(digits)
			if totalLen < width {
				padding := strings.Repeat("0", width-totalLen)
				return prefix + padding + digits
			}
			return prefix + digits
		}
	}

	goSpec := spec
	if !hasHash {
		goSpec = strings.Replace(goSpec, "#", "", 1)
	}
	goSpec += string(conv)
	result := fmt.Sprintf(goSpec, val)
	// Go's fmt produces "" for %#.0o with value 0, but C printf produces "0".
	// The # flag for octal means "ensure leading zero", so fix this.
	if conv == 'o' && hasHash && val == 0 && !strings.Contains(result, "0") {
		// Replace the empty digit portion with "0", preserving any padding
		width, leftAlign := parseFormatWidth(spec)
		if width > 1 {
			if leftAlign {
				result = "0" + strings.Repeat(" ", width-1)
			} else {
				result = strings.Repeat(" ", width-1) + "0"
			}
		} else {
			result = "0"
		}
	}
	return result
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
func formatHexFloat(f float64, prec int, hashFlag ...bool) string {
	forceDecimal := len(hashFlag) > 0 && hashFlag[0]
	// Handle subnormal floats: Go normalizes them (e.g., 0x1p-1074)
	// but C/Lua uses denormalized form (e.g., 0x0.0000000000001p-1022).
	bits := math.Float64bits(f)
	biasedExp := int((bits >> 52) & 0x7ff)
	mantissa := bits & ((1 << 52) - 1)
	if biasedExp == 0 && mantissa != 0 {
		return formatSubnormalHexFloat(f, mantissa, prec, forceDecimal)
	}

	// Get the full-precision representation from Go
	full := strconv.FormatFloat(f, 'x', -1, 64)
	full = normalizeHexExponent(full)

	if prec < 0 {
		// With # flag, force a decimal point even when there are no fractional digits
		if forceDecimal && !strings.Contains(full, ".") {
			if pIdx := strings.IndexByte(full, 'p'); pIdx >= 0 {
				full = full[:pIdx] + "." + full[pIdx:]
			}
		}
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
	if prec > 0 || forceDecimal {
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

// formatSubnormalHexFloat formats a subnormal float64 in denormalized form
// with exponent -1022, matching C/Lua 5.4 output.
func formatSubnormalHexFloat(f float64, mantissa uint64, prec int, forceDecimal bool) string {
	neg := math.Signbit(f)
	// Subnormal: leading digit is 0, mantissa has 13 hex digits (52 bits)
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteString("0x0.")

	// Format 52-bit mantissa as 13 hex digits
	hexDigits := fmt.Sprintf("%013x", mantissa)

	if prec < 0 {
		// Default: strip trailing zeros
		hexDigits = strings.TrimRight(hexDigits, "0")
		if hexDigits == "" {
			hexDigits = "0"
		}
		b.WriteString(hexDigits)
	} else if prec == 0 {
		// No fractional digits — but we still need to round
		b.Reset()
		if neg {
			b.WriteByte('-')
		}
		if forceDecimal {
			b.WriteString("0x0.")
		} else {
			b.WriteString("0x0")
		}
		// Check if we need to round up (first hex digit >= 8)
		if len(hexDigits) > 0 && hexCharToInt(hexDigits[0]) >= 8 {
			b.Reset()
			if neg {
				b.WriteByte('-')
			}
			if forceDecimal {
				b.WriteString("0x1.")
			} else {
				b.WriteString("0x1")
			}
		}
	} else {
		// Specific precision: pad or truncate with rounding
		digits := make([]int, len(hexDigits))
		for i, c := range hexDigits {
			digits[i] = hexCharToInt(byte(c))
		}
		for len(digits) <= prec {
			digits = append(digits, 0)
		}
		if prec < len(digits) {
			roundDigit := digits[prec]
			digits = digits[:prec]
			if roundDigit >= 8 {
				carry := 1
				for i := len(digits) - 1; i >= 0 && carry > 0; i-- {
					digits[i] += carry
					if digits[i] > 15 {
						digits[i] = 0
					} else {
						carry = 0
					}
				}
			}
		}
		for _, d := range digits[:prec] {
			b.WriteByte(intToHexChar(d))
		}
	}
	b.WriteString("p-1022")
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
	hasPlus := strings.Contains(spec, "+")
	hasSpace := strings.Contains(spec, " ")
	var token string
	if math.IsNaN(n) {
		if upper {
			token = "NAN"
		} else {
			token = "nan"
		}
		if math.Signbit(n) {
			token = "-" + token
		} else if hasPlus {
			token = "+" + token
		} else if hasSpace {
			token = " " + token
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
		if hasPlus {
			token = "+" + token
		} else if hasSpace {
			token = " " + token
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
	_, width, left, _ = parseFormatFlags(spec)
	return width, left
}

func parseFormatFlags(spec string) (zeroPad, width int, left bool, hash bool) {
	i := 1 // skip '%'
	for i < len(spec) && strings.ContainsRune("#0- +", rune(spec[i])) {
		switch spec[i] {
		case '-':
			left = true
		case '0':
			zeroPad = 1
		case '#':
			hash = true
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
	return zeroPad, width, left, hash
}

// validateFormatStructure checks that the format specifier follows the valid
// structure: %[flags][width][.precision]. Flags after width, negative precision,
// double dots, or double precision are all invalid. Matches Lua 5.4 behavior.
func validateFormatStructure(spec string, conv byte) {
	i := 1 // skip '%'
	// Phase 1: flags (only - + # 0 space)
	for i < len(spec) && strings.ContainsRune("-+ #0", rune(spec[i])) {
		i++
	}
	// Phase 2: width (digits only)
	for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
		i++
	}
	// Phase 3: optional precision (one '.' followed by digits)
	if i < len(spec) && spec[i] == '.' {
		i++
		for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
			i++
		}
	}
	// If we haven't consumed the entire spec, the structure is invalid.
	if i < len(spec) {
		panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
	}
}

// validateFormatWidthPrec panics if width or precision >= 100 (Lua 5.4 limit).
func validateFormatWidthPrec(spec string, conv byte) {
	// Check overall spec length (Lua 5.4 MAX_FORMAT = 32; at most 20 chars
	// of flags/width/precision between '%' and the conversion character).
	// spec here is '%' + flags/width/precision (no conversion char yet),
	// so max len(spec) is 21.
	if len(spec) > 21 {
		panic("invalid format (too long)")
	}

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
			panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
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
				panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
			}
		}
	}
}

// validateConversion checks flag/conversion compatibility per Lua 5.4 rules.
func validateConversion(spec string, conv byte) {
	// Parse only the actual flags (before width/precision digits)
	j := 1 // skip '%'
	flags := ""
	for j < len(spec) && strings.ContainsRune("#0- +", rune(spec[j])) {
		flags += string(spec[j])
		j++
	}
	// Skip width digits
	for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
		j++
	}
	// Check for precision
	hasDot := j < len(spec) && spec[j] == '.'
	hasModifiers := len(spec) > 1

	// Lua 5.4 valid flags per conversion:
	//   a/A/e/E/f/F/g/G: "-+#0 " (all flags), precision allowed
	//   o/x/X:           "-#0"   (no + or space), precision allowed
	//   d/i:             "-+0 "  (no #), precision allowed
	//   u:               "-0"    (no +, space, or #), precision allowed
	//   c/p/s:           "-"     only
	//   c/p: no precision; s: precision allowed
	switch conv {
	case 'c':
		if hasDot || strings.ContainsAny(flags, "0#+ ") {
			panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
		}
	case 'q':
		if hasModifiers {
			panic("specifier '%q' cannot have modifiers")
		}
	case 's':
		if strings.ContainsAny(flags, "0#+ ") {
			panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
		}
	case 'o', 'x', 'X':
		if strings.ContainsAny(flags, "+ ") {
			panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
		}
	case 'd', 'i':
		if strings.Contains(flags, "#") {
			panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
		}
	case 'u':
		if strings.ContainsAny(flags, "#+ ") {
			panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
		}
	case 'p':
		if hasDot || strings.ContainsAny(flags, "0#+ ") {
			panic(fmt.Sprintf("invalid conversion specification: '%s%c'", spec, conv))
		}
	case 'F':
		panic(fmt.Sprintf("invalid conversion '%s%c' to 'format'", spec, conv))
	}
}

// formatStringByBytes formats a string with byte-based width and precision,
// matching Lua 5.4's C sprintf behavior (not Go's rune-based formatting).
func formatStringByBytes(spec, str string) string {
	// Parse width and precision from spec (e.g. "%-10.5")
	s := spec[1:] // skip '%'
	leftAlign := false
	if len(s) > 0 && s[0] == '-' {
		leftAlign = true
		s = s[1:]
	}
	width := 0
	for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		width = width*10 + int(s[0]-'0')
		s = s[1:]
	}
	prec := -1
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
		prec = 0
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			prec = prec*10 + int(s[0]-'0')
			s = s[1:]
		}
	}
	// Apply precision (truncate bytes)
	if prec >= 0 && prec < len(str) {
		str = str[:prec]
	}
	// Apply width (pad with spaces, counting bytes)
	if width > len(str) {
		pad := strings.Repeat(" ", width-len(str))
		if leftAlign {
			return str + pad
		}
		return pad + str
	}
	return str
}

// luaQuote implements Lua's %q format for proper Lua-parseable quoting.
func luaQuote(v *vm.VM, val vm.Value, argIdx int) string {
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
		return formatHexFloat(f, -1)
	}
	if val.IsInt() {
		i := val.AsInt()
		if i == math.MinInt64 {
			return "0x8000000000000000"
		}
		return fmt.Sprintf("%d", i)
	}
	if !val.IsString() {
		callerArgError(v, argIdx, "string.format", "value has no literal form")
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
			// Control character (C locale iscntrl): use decimal escape.
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
	return val.PointerString()
}
