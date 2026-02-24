// Package token defines Lua token types, keywords, and source positions.
package token

import "fmt"

// Type represents a Lua token type.
type Type int

const (
	// Single-character tokens use their rune value directly (e.g. '+' = 43).
	// Multi-character and keyword tokens start at 256 to avoid collisions.

	// Keywords (ORDER RESERVED — must match Lua 5.5 order)
	AND Type = iota + 256
	BREAK
	DO
	ELSE
	ELSEIF
	END
	FALSE
	FOR
	FUNCTION
	GLOBAL // Lua 5.5
	GOTO
	IF
	IN
	LOCAL
	NIL
	NOT
	OR
	REPEAT
	RETURN
	THEN
	TRUE
	UNTIL
	WHILE

	// Multi-character operators and symbols
	IDIV    // //
	CONCAT  // ..
	DOTS    // ...
	EQ      // ==
	GE      // >=
	LE      // <=
	NE      // ~=
	SHL     // <<
	SHR     // >>
	DBCOLON // ::

	// Literals and special
	EOS    // end of source
	FLOAT  // floating-point number literal
	INT    // integer number literal
	NAME   // identifier
	STRING // string literal
)

// keywords maps keyword strings to their token types.
var keywords = map[string]Type{
	"and":      AND,
	"break":    BREAK,
	"do":       DO,
	"else":     ELSE,
	"elseif":   ELSEIF,
	"end":      END,
	"false":    FALSE,
	"for":      FOR,
	"function": FUNCTION,
	"global":   GLOBAL,
	"goto":     GOTO,
	"if":       IF,
	"in":       IN,
	"local":    LOCAL,
	"nil":      NIL,
	"not":      NOT,
	"or":       OR,
	"repeat":   REPEAT,
	"return":   RETURN,
	"then":     THEN,
	"true":     TRUE,
	"until":    UNTIL,
	"while":    WHILE,
}

// LookupIdent returns the keyword token type for ident if it is a keyword,
// or NAME otherwise.
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return NAME
}

// IsKeyword reports whether typ is a keyword token.
func (typ Type) IsKeyword() bool {
	return typ >= AND && typ <= WHILE
}

// tokenNames maps non-keyword token types to display strings.
var tokenNames = map[Type]string{
	IDIV:    "//",
	CONCAT:  "..",
	DOTS:    "...",
	EQ:      "==",
	GE:      ">=",
	LE:      "<=",
	NE:      "~=",
	SHL:     "<<",
	SHR:     ">>",
	DBCOLON: "::",
	EOS:     "<eof>",
	FLOAT:   "<number>",
	INT:     "<integer>",
	NAME:    "<name>",
	STRING:  "<string>",
}

// String returns a human-readable representation of the token type.
func (typ Type) String() string {
	// Keywords
	for kw, t := range keywords {
		if t == typ {
			return kw
		}
	}
	// Named multi-char tokens
	if name, ok := tokenNames[typ]; ok {
		return name
	}
	// Single-character tokens
	if typ > 0 && typ < 256 {
		return fmt.Sprintf("%c", rune(typ))
	}
	return fmt.Sprintf("token(%d)", int(typ))
}

// Pos represents a position in source code.
type Pos struct {
	Source string // source name (e.g. filename)
	Line   int    // 1-based line number
	Column int    // 1-based column number
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Source, p.Line, p.Column)
}

// PosError is an error with an associated source position.
type PosError struct {
	Pos Pos
	Msg string
}

func (e *PosError) Error() string { return fmt.Sprintf("%s: %s", e.Pos, e.Msg) }

// Token is a single lexical token with its type, value, and position.
type Token struct {
	Type    Type
	Literal string  // raw text of the token
	IntVal  int64   // value if Type == INT
	FltVal  float64 // value if Type == FLOAT
	Pos     Pos
}

func (t Token) String() string {
	switch t.Type {
	case NAME, STRING:
		return fmt.Sprintf("%s(%q)", t.Type, t.Literal)
	case INT:
		return fmt.Sprintf("<integer>(%s)", t.Literal)
	case FLOAT:
		return fmt.Sprintf("<number>(%s)", t.Literal)
	case EOS:
		return "<eof>"
	default:
		return t.Type.String()
	}
}
