// Package lexer implements a Lua 5.5 lexical scanner.
package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/iceisfun/golua/token"
)

const eof = -1

// Lexer tokenizes Lua source code.
type Lexer struct {
	source  string // source name (for error messages)
	input   string // full source text
	pos     int    // current byte position in input
	line    int    // current line number (1-based)
	col     int    // current column number (1-based)
	current rune   // current character (eof at end)
}

// New creates a new Lexer for the given source text.
// sourceName is used in error messages (e.g. a filename).
func New(source, input string) *Lexer {
	l := &Lexer{
		source: source,
		input:  input,
		pos:    0,
		line:   1,
		col:    1,
	}
	l.readChar()
	// Match Lua: skip the entire first line if input starts with '#'.
	// This handles Unix shebangs (#!/usr/bin/lua).
	if l.current == '#' {
		for l.current != eof && !isNewline(l.current) {
			l.readChar()
		}
		if isNewline(l.current) {
			l.incLine()
		}
	}
	return l
}

// readChar advances the lexer by one character.
func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.current = eof
		return
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.current = r
	l.pos += size
	l.col++
}

// peek returns the next character without consuming it.
func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return eof
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

// currentPos returns the current source position (pointing at current char).
func (l *Lexer) currentPos() token.Pos {
	return token.Pos{Source: l.source, Line: l.line, Column: l.col - 1}
}

// errorf creates a lexer error at the current position.
func (l *Lexer) errorf(format string, args ...any) error {
	pos := l.currentPos()
	return fmt.Errorf("%s: %s", pos, fmt.Sprintf(format, args...))
}

// incLine handles newline sequences: \n, \r, \n\r, \r\n.
func (l *Lexer) incLine() {
	old := l.current
	l.readChar() // skip \n or \r
	if isNewline(l.current) && l.current != old {
		l.readChar() // skip \n\r or \r\n
	}
	l.line++
	l.col = 1
}

// skipWhitespace skips spaces, tabs, form feeds, and vertical tabs.
// Newlines are handled separately because they affect line counting.
func (l *Lexer) skipWhitespace() {
	for {
		switch l.current {
		case ' ', '\t', '\f', '\v':
			l.readChar()
		default:
			return
		}
	}
}

// Tokenize scans the entire input and returns all tokens (ending with EOS).
func (l *Lexer) Tokenize() ([]token.Token, error) {
	var tokens []token.Token
	for {
		tok, err := l.Next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == token.EOS {
			break
		}
	}
	return tokens, nil
}

// Next returns the next token from the input.
func (l *Lexer) Next() (token.Token, error) {
	for {
		l.skipWhitespace()
		pos := l.currentPos()

		switch {
		case l.current == eof:
			return token.Token{Type: token.EOS, Pos: pos}, nil

		case isNewline(l.current):
			l.incLine()
			continue

		case l.current == '-':
			return l.scanMinus(pos)

		case l.current == '[':
			return l.scanBracketOrLongString(pos)

		case l.current == '=':
			return l.scanDoubleOp(pos, '=', token.EQ)

		case l.current == '<':
			return l.scanLessThan(pos)

		case l.current == '>':
			return l.scanGreaterThan(pos)

		case l.current == '/':
			return l.scanDoubleOp(pos, '/', token.IDIV)

		case l.current == '~':
			return l.scanDoubleOp(pos, '=', token.NE)

		case l.current == ':':
			return l.scanDoubleOp(pos, ':', token.DBCOLON)

		case l.current == '"', l.current == '\'':
			return l.scanString(pos)

		case l.current == '.':
			return l.scanDot(pos)

		case isDigit(l.current):
			return l.scanNumber(pos)

		case isAlpha(l.current):
			return l.scanIdentifier(pos)

		default:
			// Single-character token
			ch := l.current
			l.readChar()
			return token.Token{
				Type:    token.Type(ch),
				Literal: string(ch),
				Pos:     pos,
			}, nil
		}
	}
}

// scanMinus handles '-' and '--' (comments).
func (l *Lexer) scanMinus(pos token.Pos) (token.Token, error) {
	l.readChar() // skip first '-'
	if l.current != '-' {
		return token.Token{Type: token.Type('-'), Literal: "-", Pos: pos}, nil
	}
	// Comment
	l.readChar() // skip second '-'
	if l.current == '[' {
		sep := l.scanSep()
		if sep >= 2 {
			// Long comment
			if err := l.scanLongString(nil, sep); err != nil {
				return token.Token{}, err
			}
			return l.Next() // skip comment, get next token
		}
	}
	// Short comment: skip to end of line
	for l.current != eof && !isNewline(l.current) {
		l.readChar()
	}
	return l.Next()
}

// scanSep reads a separator sequence [=*[ or ]=*] and returns the count+2
// if well-formed, 1 if a single bracket, 0 if malformed.
// Does NOT consume the final bracket.
func (l *Lexer) scanSep() int {
	s := l.current
	count := 0
	l.readChar() // skip [ or ]
	for l.current == '=' {
		count++
		l.readChar()
	}
	if l.current == s {
		return count + 2
	}
	if count == 0 {
		return 1
	}
	return 0
}

// scanLongString reads a long string/comment body. If buf is non-nil,
// the content (minus delimiters) is collected into it.
func (l *Lexer) scanLongString(buf *strings.Builder, sep int) error {
	startLine := l.line
	l.readChar() // skip second bracket of opening

	// Skip initial newline if present
	if isNewline(l.current) {
		l.incLine()
	}

	for {
		switch l.current {
		case eof:
			kind := "string"
			if buf == nil {
				kind = "comment"
			}
			return fmt.Errorf("%s:%d:1: unfinished long %s (starting at line %d)",
				l.source, l.line, kind, startLine)

		case ']':
			closeSep := l.scanSep()
			if closeSep == sep {
				l.readChar() // skip final bracket
				return nil
			}
			// Not a matching close; add the characters we consumed
			if buf != nil {
				buf.WriteByte(']')
				for i := 0; i < closeSep-2; i++ {
					buf.WriteByte('=')
				}
				if closeSep == 0 {
					// malformed: scanSep consumed some '=' but didn't match
					// the chars are already consumed, just continue
				}
			}

		case '\n', '\r':
			if buf != nil {
				buf.WriteByte('\n')
			}
			l.incLine()

		default:
			if buf != nil {
				buf.WriteRune(l.current)
			}
			l.readChar()
		}
	}
}

// scanBracketOrLongString handles '[', '[[', '[=[' etc.
func (l *Lexer) scanBracketOrLongString(pos token.Pos) (token.Token, error) {
	sep := l.scanSep()
	if sep >= 2 {
		var buf strings.Builder
		if err := l.scanLongString(&buf, sep); err != nil {
			return token.Token{}, err
		}
		return token.Token{
			Type:    token.STRING,
			Literal: buf.String(),
			Pos:     pos,
		}, nil
	}
	if sep == 0 {
		return token.Token{}, l.errorf("invalid long string delimiter")
	}
	// sep == 1: just a plain '['
	return token.Token{Type: token.Type('['), Literal: "[", Pos: pos}, nil
}

// scanDoubleOp handles two-character operators where the second char
// determines if it's the double version (e.g. '==' vs '=').
func (l *Lexer) scanDoubleOp(pos token.Pos, second rune, doubleType token.Type) (token.Token, error) {
	first := l.current
	l.readChar()
	if l.current == second {
		l.readChar()
		return token.Token{
			Type:    doubleType,
			Literal: string(first) + string(second),
			Pos:     pos,
		}, nil
	}
	return token.Token{
		Type:    token.Type(first),
		Literal: string(first),
		Pos:     pos,
	}, nil
}

// scanLessThan handles '<', '<=', '<<'.
func (l *Lexer) scanLessThan(pos token.Pos) (token.Token, error) {
	l.readChar() // skip '<'
	switch l.current {
	case '=':
		l.readChar()
		return token.Token{Type: token.LE, Literal: "<=", Pos: pos}, nil
	case '<':
		l.readChar()
		return token.Token{Type: token.SHL, Literal: "<<", Pos: pos}, nil
	default:
		return token.Token{Type: token.Type('<'), Literal: "<", Pos: pos}, nil
	}
}

// scanGreaterThan handles '>', '>=', '>>'.
func (l *Lexer) scanGreaterThan(pos token.Pos) (token.Token, error) {
	l.readChar() // skip '>'
	switch l.current {
	case '=':
		l.readChar()
		return token.Token{Type: token.GE, Literal: ">=", Pos: pos}, nil
	case '>':
		l.readChar()
		return token.Token{Type: token.SHR, Literal: ">>", Pos: pos}, nil
	default:
		return token.Token{Type: token.Type('>'), Literal: ">", Pos: pos}, nil
	}
}

// scanDot handles '.', '..', '...', and numbers starting with '.'.
func (l *Lexer) scanDot(pos token.Pos) (token.Token, error) {
	l.readChar() // skip first '.'
	if l.current == '.' {
		l.readChar() // skip second '.'
		if l.current == '.' {
			l.readChar() // skip third '.'
			return token.Token{Type: token.DOTS, Literal: "...", Pos: pos}, nil
		}
		return token.Token{Type: token.CONCAT, Literal: "..", Pos: pos}, nil
	}
	if isDigit(l.current) {
		return l.scanNumberAfterDot(pos)
	}
	return token.Token{Type: token.Type('.'), Literal: ".", Pos: pos}, nil
}

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
			ch, err := l.scanEscape()
			if err != nil {
				return token.Token{}, err
			}
			if ch != -2 { // -2 is sentinel for \z (skip whitespace, write nothing)
				buf.WriteRune(ch)
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
// Returns the decoded rune.
func (l *Lexer) scanEscape() (rune, error) {
	l.readChar() // skip '\\'
	ch := l.current
	switch ch {
	case 'a':
		l.readChar()
		return '\a', nil
	case 'b':
		l.readChar()
		return '\b', nil
	case 'f':
		l.readChar()
		return '\f', nil
	case 'n':
		l.readChar()
		return '\n', nil
	case 'r':
		l.readChar()
		return '\r', nil
	case 't':
		l.readChar()
		return '\t', nil
	case 'v':
		l.readChar()
		return '\v', nil
	case '\\':
		l.readChar()
		return '\\', nil
	case '"':
		l.readChar()
		return '"', nil
	case '\'':
		l.readChar()
		return '\'', nil
	case '\n', '\r':
		l.incLine()
		return '\n', nil
	case 'x':
		return l.scanHexEscape()
	case 'u':
		return l.scanUTF8Escape()
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
		return -2, nil // sentinel: caller should not write this
	case eof:
		return 0, l.errorf("unfinished string")
	default:
		if isDigit(ch) {
			return l.scanDecimalEscape()
		}
		return 0, l.errorf("invalid escape sequence '\\%c'", ch)
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

	// Check for 0x / 0X prefix
	if l.current == '0' {
		buf.WriteRune(l.current)
		l.readChar()
		if l.current == 'x' || l.current == 'X' {
			isHex = true
			buf.WriteRune(l.current)
			l.readChar()
		}
	}

	return l.scanNumberBody(pos, &buf, isHex)
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
		return token.Token{}, fmt.Errorf("%s: malformed number '%s%c'",
			pos, buf.String(), l.current)
	}

	raw := buf.String()

	if isFloat {
		val, err := parseFloat(raw)
		if err != nil {
			return token.Token{}, fmt.Errorf("%s: malformed number '%s'", pos, raw)
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
			return token.Token{}, fmt.Errorf("%s: malformed number '%s'", pos, raw)
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

// Helper functions

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
