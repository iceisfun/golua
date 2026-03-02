package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/iceisfun/golua/token"
)

// scanString reads a short string delimited by ' or ".
func (l *Lexer) scanString(pos token.Pos) (token.Token, error) {
	delim := l.current
	l.readChar() // skip opening delimiter

	var buf strings.Builder
	for l.current != delim {
		switch l.current {
		case eof:
			return token.Token{}, l.errorf("unfinished string")
		case '\n', '\r':
			return token.Token{}, l.errorf("unfinished string")
		case '\\':
			ch, isUnicode, err := l.scanEscape()
			if err != nil {
				return token.Token{}, err
			}
			if ch != -2 { // -2 is sentinel for \z (skip whitespace, write nothing)
				if isUnicode {
					// \u{XXXX} always produces UTF-8 encoding.
					// Lua allows codepoints up to 0x7FFFFFFF (extended UTF-8),
					// beyond Go's Unicode limit of 0x10FFFF.
					writeExtendedUTF8(&buf, ch)
				} else if ch <= 0xFF {
					// Byte-level escapes (\xNN, \DDD, \a, \n, etc.)
					// must produce exact byte values, not UTF-8 re-encoding.
					buf.WriteByte(byte(ch))
				} else {
					buf.WriteRune(ch)
				}
			}
		default:
			buf.WriteRune(l.current)
			l.readChar()
		}
	}
	l.readChar() // skip closing delimiter

	return token.Token{
		Type:    token.STRING,
		Literal: buf.String(),
		Pos:     pos,
	}, nil
}

// scanEscape reads an escape sequence after '\'.
// Returns the decoded rune and whether it is a unicode (\u{}) escape.
// Unicode escapes must always be written as UTF-8 (via WriteRune),
// while byte escapes (\xNN, \DDD) must be written as raw bytes.
func (l *Lexer) scanEscape() (rune, bool, error) {
	l.readChar() // skip '\\'
	ch := l.current
	switch ch {
	case 'a':
		l.readChar()
		return '\a', false, nil
	case 'b':
		l.readChar()
		return '\b', false, nil
	case 'f':
		l.readChar()
		return '\f', false, nil
	case 'n':
		l.readChar()
		return '\n', false, nil
	case 'r':
		l.readChar()
		return '\r', false, nil
	case 't':
		l.readChar()
		return '\t', false, nil
	case 'v':
		l.readChar()
		return '\v', false, nil
	case '\\':
		l.readChar()
		return '\\', false, nil
	case '"':
		l.readChar()
		return '"', false, nil
	case '\'':
		l.readChar()
		return '\'', false, nil
	case '\n', '\r':
		l.incLine()
		return '\n', false, nil
	case 'x':
		r, err := l.scanHexEscape()
		return r, false, err
	case 'u':
		r, err := l.scanUTF8Escape()
		return r, true, err
	case 'z':
		// Skip whitespace
		l.readChar() // skip 'z'
		for l.current != eof && isWhitespace(l.current) {
			if isNewline(l.current) {
				l.incLine()
			} else {
				l.readChar()
			}
		}
		return -2, false, nil // sentinel: caller should not write this
	case eof:
		return 0, false, l.errorf("unfinished string")
	default:
		if isDigit(ch) {
			r, err := l.scanDecimalEscape()
			return r, false, err
		}
		return 0, false, l.errorf("invalid escape sequence '\\%c'", ch)
	}
}

// scanHexEscape reads \xNN.
func (l *Lexer) scanHexEscape() (rune, error) {
	l.readChar() // skip 'x'
	d1, ok := hexVal(l.current)
	if !ok {
		return 0, l.errorf("hexadecimal digit expected")
	}
	l.readChar()
	d2, ok := hexVal(l.current)
	if !ok {
		return 0, l.errorf("hexadecimal digit expected")
	}
	l.readChar()
	return rune(d1*16 + d2), nil
}

// scanUTF8Escape reads \u{XXXX}.
func (l *Lexer) scanUTF8Escape() (rune, error) {
	l.readChar() // skip 'u'
	if l.current != '{' {
		return 0, l.errorf("missing '{'")
	}
	l.readChar() // skip '{'
	val, ok := hexVal(l.current)
	if !ok {
		return 0, l.errorf("hexadecimal digit expected")
	}
	l.readChar()
	for l.current != '}' {
		if l.current == eof {
			return 0, l.errorf("missing '}'")
		}
		d, ok := hexVal(l.current)
		if !ok {
			return 0, l.errorf("missing '}'")
		}
		val = val*16 + d
		if val > 0x7FFFFFFF {
			return 0, l.errorf("UTF-8 value too large")
		}
		l.readChar()
	}
	l.readChar() // skip '}'
	return rune(val), nil
}

// writeExtendedUTF8 encodes a codepoint as extended UTF-8 (up to 6 bytes),
// supporting Lua's full range of 0x00–0x7FFFFFFF. Go's WriteRune only handles
// up to 0x10FFFF; values beyond that are replaced with U+FFFD.
func writeExtendedUTF8(buf *strings.Builder, r rune) {
	cp := uint32(r)
	switch {
	case cp <= 0x7F:
		buf.WriteByte(byte(cp))
	case cp <= 0x7FF:
		buf.WriteByte(byte(0xC0 | (cp >> 6)))
		buf.WriteByte(byte(0x80 | (cp & 0x3F)))
	case cp <= 0xFFFF:
		buf.WriteByte(byte(0xE0 | (cp >> 12)))
		buf.WriteByte(byte(0x80 | ((cp >> 6) & 0x3F)))
		buf.WriteByte(byte(0x80 | (cp & 0x3F)))
	case cp <= 0x1FFFFF:
		buf.WriteByte(byte(0xF0 | (cp >> 18)))
		buf.WriteByte(byte(0x80 | ((cp >> 12) & 0x3F)))
		buf.WriteByte(byte(0x80 | ((cp >> 6) & 0x3F)))
		buf.WriteByte(byte(0x80 | (cp & 0x3F)))
	case cp <= 0x3FFFFFF:
		buf.WriteByte(byte(0xF8 | (cp >> 24)))
		buf.WriteByte(byte(0x80 | ((cp >> 18) & 0x3F)))
		buf.WriteByte(byte(0x80 | ((cp >> 12) & 0x3F)))
		buf.WriteByte(byte(0x80 | ((cp >> 6) & 0x3F)))
		buf.WriteByte(byte(0x80 | (cp & 0x3F)))
	default: // up to 0x7FFFFFFF
		buf.WriteByte(byte(0xFC | (cp >> 30)))
		buf.WriteByte(byte(0x80 | ((cp >> 24) & 0x3F)))
		buf.WriteByte(byte(0x80 | ((cp >> 18) & 0x3F)))
		buf.WriteByte(byte(0x80 | ((cp >> 12) & 0x3F)))
		buf.WriteByte(byte(0x80 | ((cp >> 6) & 0x3F)))
		buf.WriteByte(byte(0x80 | (cp & 0x3F)))
	}
}

// scanDecimalEscape reads \ddd (up to 3 decimal digits).
func (l *Lexer) scanDecimalEscape() (rune, error) {
	val := int(l.current - '0')
	l.readChar()
	for i := 1; i < 3 && isDigit(l.current); i++ {
		val = val*10 + int(l.current-'0')
		l.readChar()
	}
	if val > 255 {
		return 0, l.errorf("decimal escape too large")
	}
	return rune(val), nil
}

// scanNumber reads a numeric literal.
func (l *Lexer) scanNumber(pos token.Pos) (token.Token, error) {
	var buf strings.Builder
	isHex := false

	// Check for 0x / 0X / 0b / 0B prefix
	if l.current == '0' {
		buf.WriteRune(l.current)
		l.readChar()
		if l.current == 'x' || l.current == 'X' {
			isHex = true
			buf.WriteRune(l.current)
			l.readChar()
		} else if l.current == 'b' || l.current == 'B' {
			return l.scanBinaryNumber(pos, &buf)
		}
	}

	return l.scanNumberBody(pos, &buf, isHex)
}

// scanBinaryNumber reads a binary integer literal after "0b" or "0B".
func (l *Lexer) scanBinaryNumber(pos token.Pos, buf *strings.Builder) (token.Token, error) {
	buf.WriteRune(l.current) // write 'b' or 'B'
	l.readChar()

	// Must have at least one binary digit.
	if l.current != '0' && l.current != '1' {
		if isAlpha(l.current) || isDigit(l.current) {
			return token.Token{}, &token.PosError{Pos: pos,
				Msg: fmt.Sprintf("malformed number '%s%c'", buf.String(), l.current)}
		}
		return token.Token{}, &token.PosError{Pos: pos,
			Msg: fmt.Sprintf("malformed number '%s'", buf.String())}
	}

	for l.current == '0' || l.current == '1' {
		buf.WriteRune(l.current)
		l.readChar()
	}

	// Reject trailing digits or letters (e.g. 0b102, 0b10abc).
	if isAlpha(l.current) || isDigit(l.current) {
		return token.Token{}, &token.PosError{Pos: pos,
			Msg: fmt.Sprintf("malformed number '%s%c'", buf.String(), l.current)}
	}

	raw := buf.String()
	ival, err := parseInt(raw)
	if err != nil {
		return token.Token{}, &token.PosError{Pos: pos,
			Msg: fmt.Sprintf("malformed number '%s'", raw)}
	}
	return token.Token{
		Type:    token.INT,
		Literal: raw,
		IntVal:  ival,
		Pos:     pos,
	}, nil
}

// scanNumberAfterDot reads a number that started with '.'.
func (l *Lexer) scanNumberAfterDot(pos token.Pos) (token.Token, error) {
	var buf strings.Builder
	buf.WriteByte('.')
	return l.scanNumberBody(pos, &buf, false)
}

// scanNumberBody reads the rest of a number literal.
func (l *Lexer) scanNumberBody(pos token.Pos, buf *strings.Builder, isHex bool) (token.Token, error) {
	expo := "eE"
	if isHex {
		expo = "pP"
	}

	isFloat := strings.Contains(buf.String(), ".")

	for {
		if l.current != eof && strings.ContainsRune(expo, l.current) {
			isFloat = true
			buf.WriteRune(l.current)
			l.readChar()
			if l.current == '+' || l.current == '-' {
				buf.WriteRune(l.current)
				l.readChar()
			}
		} else if isHexDigit(l.current) || l.current == '.' {
			if l.current == '.' {
				isFloat = true
			}
			buf.WriteRune(l.current)
			l.readChar()
		} else {
			break
		}
	}

	// Check for letter touching numeral (e.g. "123abc")
	if isAlpha(l.current) {
		return token.Token{}, &token.PosError{Pos: pos,
			Msg: fmt.Sprintf("malformed number '%s%c'", buf.String(), l.current)}
	}

	raw := buf.String()

	if isFloat {
		val, err := parseFloat(raw)
		if err != nil {
			return token.Token{}, &token.PosError{Pos: pos,
				Msg: fmt.Sprintf("malformed number '%s'", raw)}
		}
		return token.Token{
			Type:    token.FLOAT,
			Literal: raw,
			FltVal:  val,
			Pos:     pos,
		}, nil
	}

	ival, err := parseInt(raw)
	if err != nil {
		// Integer overflow: fall back to float (matches Lua behaviour).
		fval, ferr := parseFloat(raw)
		if ferr != nil {
			return token.Token{}, &token.PosError{Pos: pos,
				Msg: fmt.Sprintf("malformed number '%s'", raw)}
		}
		return token.Token{
			Type:    token.FLOAT,
			Literal: raw,
			FltVal:  fval,
			Pos:     pos,
		}, nil
	}
	return token.Token{
		Type:    token.INT,
		Literal: raw,
		IntVal:  ival,
		Pos:     pos,
	}, nil
}

// scanIdentifier reads an identifier or keyword.
func (l *Lexer) scanIdentifier(pos token.Pos) (token.Token, error) {
	var buf strings.Builder
	for isAlpha(l.current) || isDigit(l.current) {
		buf.WriteRune(l.current)
		l.readChar()
	}
	name := buf.String()
	typ := token.LookupIdent(name)
	return token.Token{
		Type:    typ,
		Literal: name,
		Pos:     pos,
	}, nil
}

// ---------------------------------------------------------------------------
// Character predicates
// ---------------------------------------------------------------------------

func isNewline(r rune) bool {
	return r == '\n' || r == '\r'
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\f' || r == '\v' || r == '\n' || r == '\r'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexDigit(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || unicode.IsLetter(r)
}

// ---------------------------------------------------------------------------
// Number parsing helpers
// ---------------------------------------------------------------------------

func hexVal(r rune) (int, bool) {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0'), true
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10, true
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10, true
	default:
		return 0, false
	}
}

func parseFloat(s string) (float64, error) {
	// Hex numbers need a 'p' exponent for Go's ParseFloat.
	// If it has 0x prefix but no 'p'/'P', append "p0".
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "0x") && !strings.ContainsAny(lower, "pP") {
		s = s + "p0"
	}
	return strconv.ParseFloat(s, 64)
}

func parseInt(s string) (int64, error) {
	// Handle hex — use ParseUint to handle full 64-bit range (e.g. 0xFFFFFFFFFFFFFFFF),
	// then reinterpret as int64 (matching Lua's wrapping behaviour).
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		u, err := strconv.ParseUint(s[2:], 16, 64)
		if err != nil {
			return 0, err
		}
		return int64(u), nil
	}
	// Handle binary — same approach as hex.
	if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		u, err := strconv.ParseUint(s[2:], 2, 64)
		if err != nil {
			return 0, err
		}
		return int64(u), nil
	}
	// Decimal: try ParseInt first; on overflow try ParseUint.
	v, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return v, nil
	}
	u, err2 := strconv.ParseUint(s, 10, 64)
	if err2 != nil {
		return 0, err // return original error
	}
	return int64(u), nil
}
