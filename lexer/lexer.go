// Package lexer implements a lexical scanner for Lua 5.4 source code.
//
// The scanner converts UTF-8 source text into a stream of tokens. It handles
// all Lua lexical elements including long strings ([[ ]]), nested long comments,
// escape sequences (\xNN, \u{XXXX}, \z), hex float literals, and optional
// shebang line stripping.
//
// Lua 5.4 Reference: §3.1 – Lexical Conventions.
package lexer

import (
	"fmt"
	"strings"
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
// If stripShebang is true, a leading '#' line is skipped (for file loading).
func New(source, input string, stripShebang bool) *Lexer {
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
	if stripShebang && l.current == '#' {
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
	return &token.PosError{Pos: l.currentPos(), Msg: fmt.Sprintf(format, args...)}
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
			return &token.PosError{
				Pos: token.Pos{Source: l.source, Line: l.line, Column: 1},
				Msg: fmt.Sprintf("unfinished long %s (starting at line %d)", kind, startLine),
			}

		case ']':
			// Try to match closing bracket sequence ]===...===]
			l.readChar() // skip ']'
			eqCount := 0
			for l.current == '=' {
				eqCount++
				l.readChar()
			}
			if l.current == ']' && eqCount+2 == sep {
				l.readChar() // skip final ']'
				return nil
			}
			// Not a matching close; write back the consumed characters
			if buf != nil {
				buf.WriteByte(']')
				for i := 0; i < eqCount; i++ {
					buf.WriteByte('=')
				}
				// l.current is now pointing at the non-matching char;
				// it will be handled by the next iteration of the loop
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

