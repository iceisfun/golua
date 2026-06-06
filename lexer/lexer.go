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
	"math"
	"strings"
	"unicode/utf8"

	"github.com/iceisfun/golua/v2/token"
)

const eof = -1

// maxLines is the maximum number of lines a Lua source file may contain.
// Lua 5.4 uses MAX_INT (INT_MAX). We use a lower practical limit that is
// still far beyond any real source file. This fires during lexing for
// string-based loads; reader-based loads also check during accumulation.
const maxLines = math.MaxInt32

// maxTokenLen is the maximum length of a single lexical element (identifier,
// string literal, number literal). Prevents unbounded memory growth when
// scanning pathologically large tokens. Lua 5.4 relies on allocator failure;
// GoLua uses an explicit limit since Go's runtime crashes on OOM.
const maxTokenLen = 1 << 24 // 16 MB

// Lexer tokenizes Lua source code.
type Lexer struct {
	source         string // source name (for error messages)
	input          string // full source text
	pos            int    // current byte position in input
	line           int    // current line number (1-based)
	col            int    // current column number (1-based)
	current        rune   // current character (eof at end)
	rawByte        bool   // true when current holds a raw byte (invalid UTF-8)
	stringRawStart int    // byte position of opening delimiter for string scanning (used in "near" context)
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
			_ = l.incLine() // first line; cannot exceed maxLines
		}
	}
	return l
}

// readChar advances the lexer by one character, using UTF-8 decoding for
// valid multi-byte sequences (e.g. Unicode identifiers). Invalid single
// bytes are preserved as their raw byte value (not replaced with RuneError)
// so that non-ASCII bytes in string literals pass through correctly.
// The rawByte flag is set when the current character is a raw invalid byte.
func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.current = eof
		l.rawByte = false
		return
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == utf8.RuneError && size == 1 {
		// Invalid UTF-8 byte: preserve raw byte value
		l.current = rune(l.input[l.pos])
		l.rawByte = true
	} else {
		l.current = r
		l.rawByte = false
	}
	l.pos += size
	l.col++
}

// writeCurrent writes the current character to buf. Raw bytes (invalid UTF-8)
// are written as single bytes; valid UTF-8 characters use rune encoding.
func (l *Lexer) writeCurrent(buf *strings.Builder) {
	if l.rawByte {
		buf.WriteByte(byte(l.current))
	} else {
		buf.WriteRune(l.current)
	}
}

// currentPos returns the current source position (pointing at current char).
func (l *Lexer) currentPos() token.Pos {
	return token.Pos{Source: l.source, Line: l.line, Column: l.col - 1}
}

// errorf creates a lexer error at the current position.
func (l *Lexer) errorf(format string, args ...any) error {
	return &token.PosError{Pos: l.currentPos(), Msg: fmt.Sprintf(format, args...)}
}

// stringNear returns the "near" context for an error during string scanning.
// Returns the raw source text from the opening delimiter through the current
// position, wrapped in single quotes. Matches Lua 5.4's txtToken(TK_STRING).
func (l *Lexer) stringNear() string {
	end := l.pos
	if end > len(l.input) {
		end = len(l.input)
	}
	return "'" + l.input[l.stringRawStart:end] + "'"
}

// stringNearExcludeCurrent returns the "near" context excluding the current
// character. Used for unfinished string errors where the terminating newline
// should not appear in the error message.
func (l *Lexer) stringNearExcludeCurrent() string {
	end := l.pos
	if l.current != eof {
		if l.rawByte {
			end--
		} else {
			end -= utf8.RuneLen(l.current)
		}
	}
	if end > len(l.input) {
		end = len(l.input)
	}
	if end < l.stringRawStart {
		end = l.stringRawStart
	}
	return "'" + l.input[l.stringRawStart:end] + "'"
}

// stringErrorf creates a lexer error with "near" context from the string
// being scanned. Used for escape sequence errors where the raw source text
// helps identify the problem location.
func (l *Lexer) stringErrorf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return &token.PosError{Pos: l.currentPos(), Msg: msg + " near " + l.stringNear()}
}

// finalizeStringErr constructs a PosError for a deferred escape-sequence
// error after the lexer has consumed through to the closing delimiter (or
// EOF/newline). The "near" context spans the opening delimiter through the
// current position, matching Lua 5.4/5.5's diagnostic format. If the
// pendingMsg already contains " near '", its existing near-clause is
// stripped and replaced so the final message quotes the full literal.
func (l *Lexer) finalizeStringErr(pendingMsg string) error {
	// Strip any " near '...'" suffix that was attached when the escape
	// error was first raised (stringErrorf appends one). We'll rebuild it
	// with the full literal's range below.
	if idx := strings.Index(pendingMsg, " near '"); idx >= 0 {
		pendingMsg = pendingMsg[:idx]
	}
	return &token.PosError{Pos: l.currentPos(), Msg: pendingMsg + " near " + l.stringNear()}
}

// incLine handles newline sequences: \n, \r, \n\r, \r\n.
// Returns an error if the line count exceeds maxLines (Lua 5.4 MAX_INT).
func (l *Lexer) incLine() error {
	old := l.current
	l.readChar() // skip \n or \r
	if isNewline(l.current) && l.current != old {
		l.readChar() // skip \n\r or \r\n
	}
	l.line++
	if l.line >= maxLines {
		return l.errorf("chunk has too many lines")
	}
	l.col = 1
	return nil
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
			if err := l.incLine(); err != nil {
				return token.Token{}, err
			}
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

		case !l.rawByte && isAlpha(l.current):
			return l.scanIdentifier(pos)

		default:
			// Single-character token. For multi-byte UTF-8 characters,
			// consume only the first byte (matching Lua 5.4's byte-by-byte
			// behaviour) so that error messages report the raw first byte
			// value rather than the decoded Unicode codepoint.
			ch := l.current
			if !l.rawByte && utf8.RuneLen(ch) > 1 {
				// Rewind: re-position to just after the first byte.
				firstByte := l.input[l.pos-utf8.RuneLen(ch)]
				l.pos = l.pos - utf8.RuneLen(ch) + 1
				// Re-read next character so lexer state is consistent.
				ch = rune(firstByte)
				l.current = eof // will be set by next readChar
				l.rawByte = false
				// Advance to set l.current for the next token.
				l.readChar()
				return token.Token{
					Type:    token.Type(firstByte),
					Literal: string(rune(firstByte)),
					Pos:     pos,
				}, nil
			}
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
		if err := l.incLine(); err != nil {
			return err
		}
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
				Msg: fmt.Sprintf("unfinished long %s (starting at line %d) near <eof>", kind, startLine),
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
			if err := l.incLine(); err != nil {
				return err
			}

		default:
			if buf != nil {
				l.writeCurrent(buf)
			}
			l.readChar()
		}
	}
}

// scanBracketOrLongString handles '[', '[[', '[=[' etc.
func (l *Lexer) scanBracketOrLongString(pos token.Pos) (token.Token, error) {
	rawStart := l.pos - 1 // byte position of opening '['
	sep := l.scanSep()
	if sep >= 2 {
		var buf strings.Builder
		if err := l.scanLongString(&buf, sep); err != nil {
			return token.Token{}, err
		}
		// Capture raw source text with delimiters for "near" context.
		// After scanLongString, l.pos is past the final ']' that was consumed.
		// But scanLongString's last readChar advanced past ']', so the raw
		// text from rawStart to the position before l.current covers everything.
		rawEnd := l.pos
		if l.current != eof {
			if l.rawByte {
				rawEnd--
			} else {
				rawEnd -= utf8.RuneLen(l.current)
			}
		}
		if rawEnd > len(l.input) {
			rawEnd = len(l.input)
		}
		raw := l.input[rawStart:rawEnd]
		return token.Token{
			Type:    token.STRING,
			Literal: buf.String(),
			Raw:     raw,
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
