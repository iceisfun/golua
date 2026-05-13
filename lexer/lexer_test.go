package lexer

import (
	"testing"

	"github.com/iceisfun/golua/token"
)

// helper to tokenize and check no error
func mustTokenize(t *testing.T, input string) []token.Token {
	t.Helper()
	l := New("test", input, true)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return tokens
}

// helper to check a single token (excluding final EOS)
func expectSingle(t *testing.T, input string, typ token.Type, literal string) {
	t.Helper()
	tokens := mustTokenize(t, input)
	if len(tokens) != 2 { // token + EOS
		t.Fatalf("expected 2 tokens (token + EOS), got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Type != typ {
		t.Errorf("type: want %v, got %v", typ, tokens[0].Type)
	}
	if tokens[0].Literal != literal {
		t.Errorf("literal: want %q, got %q", literal, tokens[0].Literal)
	}
}

func TestEmptyInput(t *testing.T) {
	tokens := mustTokenize(t, "")
	if len(tokens) != 1 || tokens[0].Type != token.EOS {
		t.Fatalf("expected single EOS token, got %v", tokens)
	}
}

func TestWhitespaceOnly(t *testing.T) {
	tokens := mustTokenize(t, "   \t\n\r  ")
	if len(tokens) != 1 || tokens[0].Type != token.EOS {
		t.Fatalf("expected single EOS token, got %v", tokens)
	}
}

func TestSingleCharTokens(t *testing.T) {
	cases := []struct {
		input string
		typ   token.Type
	}{
		{"+", token.Type('+')},
		{"-", token.Type('-')},
		{"*", token.Type('*')},
		{"%", token.Type('%')},
		{"^", token.Type('^')},
		{"&", token.Type('&')},
		{"|", token.Type('|')},
		{"(", token.Type('(')},
		{")", token.Type(')')},
		{"{", token.Type('{')},
		{"}", token.Type('}')},
		{"]", token.Type(']')},
		{";", token.Type(';')},
		{",", token.Type(',')},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			expectSingle(t, tc.input, tc.typ, tc.input)
		})
	}
}

// TestHashOperator tests # when not at start of input (where it's a shebang).
func TestHashOperator(t *testing.T) {
	// # after whitespace is the length operator
	expectSingle(t, " #", token.Type('#'), "#")
}

func TestShebangSkipped(t *testing.T) {
	// # at start of input skips the entire first line
	l := New("test", "#!something\nlocal x", true)
	tok, err := l.Next()
	if err != nil {
		t.Fatal(err)
	}
	if tok.Type != token.LOCAL {
		t.Errorf("expected LOCAL after shebang, got %s", tok)
	}
	if tok.Pos.Line != 2 {
		t.Errorf("expected line 2 after shebang, got line %d", tok.Pos.Line)
	}
}

func TestMultiCharOperators(t *testing.T) {
	cases := []struct {
		input   string
		typ     token.Type
		literal string
	}{
		{"==", token.EQ, "=="},
		{"~=", token.NE, "~="},
		{"<=", token.LE, "<="},
		{">=", token.GE, ">="},
		{"<<", token.SHL, "<<"},
		{">>", token.SHR, ">>"},
		{"//", token.IDIV, "//"},
		{"::", token.DBCOLON, "::"},
		{"..", token.CONCAT, ".."},
		{"...", token.DOTS, "..."},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			expectSingle(t, tc.input, tc.typ, tc.literal)
		})
	}
}

func TestOperatorAmbiguity(t *testing.T) {
	// '=' alone
	expectSingle(t, "=", token.Type('='), "=")
	// '~' alone
	expectSingle(t, "~", token.Type('~'), "~")
	// '<' alone
	expectSingle(t, "<", token.Type('<'), "<")
	// '>' alone
	expectSingle(t, ">", token.Type('>'), ">")
	// '/' alone
	expectSingle(t, "/", token.Type('/'), "/")
	// ':' alone
	expectSingle(t, ":", token.Type(':'), ":")
	// '.' alone
	expectSingle(t, ".", token.Type('.'), ".")
}

func TestDotSequences(t *testing.T) {
	// . followed by something non-numeric
	tokens := mustTokenize(t, ".abc")
	if tokens[0].Type != token.Type('.') {
		t.Errorf("expected '.', got %v", tokens[0])
	}
	if tokens[1].Type != token.NAME || tokens[1].Literal != "abc" {
		t.Errorf("expected NAME 'abc', got %v", tokens[1])
	}

	// .. followed by something
	tokens = mustTokenize(t, "..x")
	if tokens[0].Type != token.CONCAT {
		t.Errorf("expected CONCAT, got %v", tokens[0])
	}
}

func TestKeywords(t *testing.T) {
	keywords := []struct {
		kw  string
		typ token.Type
	}{
		{"and", token.AND},
		{"break", token.BREAK},
		{"do", token.DO},
		{"else", token.ELSE},
		{"elseif", token.ELSEIF},
		{"end", token.END},
		{"false", token.FALSE},
		{"for", token.FOR},
		{"function", token.FUNCTION},
		{"global", token.GLOBAL},
		{"goto", token.GOTO},
		{"if", token.IF},
		{"in", token.IN},
		{"local", token.LOCAL},
		{"nil", token.NIL},
		{"not", token.NOT},
		{"or", token.OR},
		{"repeat", token.REPEAT},
		{"return", token.RETURN},
		{"then", token.THEN},
		{"true", token.TRUE},
		{"until", token.UNTIL},
		{"while", token.WHILE},
	}
	for _, tc := range keywords {
		t.Run(tc.kw, func(t *testing.T) {
			expectSingle(t, tc.kw, tc.typ, tc.kw)
		})
	}
}

func TestIdentifiers(t *testing.T) {
	cases := []string{"foo", "bar123", "_private", "_", "__index", "camelCase"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			expectSingle(t, name, token.NAME, name)
		})
	}
}

func TestUnicodeIdentifiersRejected(t *testing.T) {
	// Lua 5.4 only accepts ASCII letters [a-zA-Z] and underscore for identifiers.
	// Unicode letters must NOT be accepted as identifier characters.
	cases := []struct {
		name  string
		input string
	}{
		{"pure_cjk", "你好"},
		{"cyrillic", "привет"},
		{"greek", "α"},
		{"accented_latin", "café"},
		{"japanese_katakana", "テスト"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := New("test", tc.input, true)
			tokens, _ := l.Tokenize()
			// Unicode characters should NOT produce a single NAME token
			for _, tok := range tokens {
				if tok.Type == token.NAME && tok.Literal == tc.input {
					t.Errorf("Unicode input %q should not be accepted as a NAME token", tc.input)
				}
			}
		})
	}
}

func TestRawNonASCIIBytesInStrings(t *testing.T) {
	// Raw non-ASCII bytes (invalid UTF-8) inside string literals must be
	// preserved as-is, not re-encoded through UTF-8 rune decoding.
	// This is critical for Lua's byte-oriented string semantics.
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lone high byte in short string",
			input: "\"\\xE3\"", // \xE3 escape → byte 0xE3
			want:  "\xE3",
		},
		{
			name:  "raw high byte in short string",
			input: "\"\xE3\"", // raw byte 0xE3 between quotes
			want:  "\xE3",
		},
		{
			name:  "raw high byte in long string",
			input: "[[\xE3]]", // raw byte 0xE3 in long string
			want:  "\xE3",
		},
		{
			name:  "multiple raw bytes in short string",
			input: "\"\x80\xBF\xFF\"",
			want:  "\x80\xBF\xFF",
		},
		{
			name:  "valid UTF-8 preserved in short string",
			input: "\"\xC3\xA0\"", // valid UTF-8 for U+00E0 (à)
			want:  "\xC3\xA0",
		},
		{
			name:  "valid multibyte UTF-8 preserved in short string",
			input: "\"\xE6\x97\xA5\"", // valid UTF-8 for 日 (U+65E5)
			want:  "\xE6\x97\xA5",
		},
		{
			name:  "valid UTF-8 preserved in long string",
			input: "[[\xC3\xA0]]",
			want:  "\xC3\xA0",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens := mustTokenize(t, tc.input)
			if len(tokens) < 2 {
				t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
			}
			got := tokens[0].Literal
			if got != tc.want {
				t.Errorf("string content mismatch\n  want bytes: %x\n  got bytes:  %x", []byte(tc.want), []byte(got))
			}
		})
	}
}

func TestRawBytesNotIdentifiers(t *testing.T) {
	// Raw non-ASCII bytes must NOT be accepted as identifier characters.
	// Only valid UTF-8 multi-byte sequences should form identifiers.
	l := New("test", "\xE3=1", true)
	tokens, err := l.Tokenize()
	if err == nil {
		// If it didn't error, the 0xE3 byte should NOT be an identifier
		for _, tok := range tokens {
			if tok.Type == token.NAME && tok.Literal == "\xe3" {
				t.Error("raw byte 0xE3 should not be accepted as an identifier")
			}
		}
	}
	// Either an error or the byte is treated as an unknown symbol — both are fine
}

func TestIntegers(t *testing.T) {
	cases := []struct {
		input string
		val   int64
	}{
		{"0", 0},
		{"42", 42},
		{"123456", 123456},
		{"0xff", 255},
		{"0xFF", 255},
		{"0x10", 16},
		{"0XDEAD", 0xDEAD},
		{"0b0", 0},
		{"0b1", 1},
		{"0b10", 2},
		{"0b1010", 10},
		{"0B1010", 10},
		{"0b1111", 15},
		{"0b11111111", 255},
		{"0b1111111111111111111111111111111111111111111111111111111111111111", -1}, // all 64 bits set
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			tokens := mustTokenize(t, tc.input)
			if tokens[0].Type != token.INT {
				t.Fatalf("expected INT, got %v", tokens[0].Type)
			}
			if tokens[0].IntVal != tc.val {
				t.Errorf("value: want %d, got %d", tc.val, tokens[0].IntVal)
			}
		})
	}
}

func TestFloats(t *testing.T) {
	cases := []struct {
		input string
		val   float64
	}{
		{"3.14", 3.14},
		{".5", 0.5},
		{"1e10", 1e10},
		{"1E10", 1e10},
		{"1.5e2", 150.0},
		{"3.14e-1", 0.314},
		{"0x1p10", 1024.0},
		{"0x1.8p1", 3.0},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			tokens := mustTokenize(t, tc.input)
			if tokens[0].Type != token.FLOAT {
				t.Fatalf("expected FLOAT, got %v", tokens[0].Type)
			}
			if tokens[0].FltVal != tc.val {
				t.Errorf("value: want %g, got %g", tc.val, tokens[0].FltVal)
			}
		})
	}
}

func TestMalformedNumber(t *testing.T) {
	l := New("test", "123abc", true)
	_, err := l.Next()
	if err == nil {
		t.Fatal("expected error for malformed number")
	}
}

func TestMalformedBinaryNumber(t *testing.T) {
	cases := []string{
		"0b",     // no digits after prefix
		"0B",     // no digits after prefix (uppercase)
		"0b102",  // non-binary digit
		"0b2",    // non-binary digit immediately
		"0b10abc", // trailing letters
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			l := New("test", input, true)
			_, err := l.Next()
			if err == nil {
				t.Fatalf("expected error for malformed binary number %q", input)
			}
		})
	}
}

func TestShortStrings(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{`""`, ""},
		{`"hello\nworld"`, "hello\nworld"},
		{`"tab\there"`, "tab\there"},
		{`"quote\""`, `quote"`},
		{`'single\'s'`, "single's"},
		{`"back\\"`, "back\\"},
		{`"\a\b\f\r\v"`, "\a\b\f\r\v"},
		{`"\x41\x42"`, "AB"},
		{`"\65\66"`, "AB"},    // decimal escapes
		{`"\u{41}"`, "A"},     // UTF-8 escape
		{`"\u{1F600}"`, "😀"}, // emoji
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			tokens := mustTokenize(t, tc.input)
			if tokens[0].Type != token.STRING {
				t.Fatalf("expected STRING, got %v", tokens[0].Type)
			}
			if tokens[0].Literal != tc.expected {
				t.Errorf("value: want %q, got %q", tc.expected, tokens[0].Literal)
			}
		})
	}
}

func TestEscapeZ(t *testing.T) {
	tokens := mustTokenize(t, `"hello\z    world"`)
	if tokens[0].Literal != "helloworld" {
		t.Errorf("\\z: want %q, got %q", "helloworld", tokens[0].Literal)
	}
}

func TestEscapeZWithNewlines(t *testing.T) {
	tokens := mustTokenize(t, "\"hello\\z\n   \n   world\"")
	if tokens[0].Literal != "helloworld" {
		t.Errorf("\\z with newlines: want %q, got %q", "helloworld", tokens[0].Literal)
	}
}

func TestStringNewlineEscape(t *testing.T) {
	tokens := mustTokenize(t, "\"hello\\\nworld\"")
	if tokens[0].Literal != "hello\nworld" {
		t.Errorf("newline escape: want %q, got %q", "hello\nworld", tokens[0].Literal)
	}
}

func TestUnfinishedString(t *testing.T) {
	l := New("test", `"hello`, true)
	_, err := l.Next()
	if err == nil {
		t.Fatal("expected error for unfinished string")
	}
}

func TestLongStrings(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic", "[[hello]]", "hello"},
		{"with equals", "[==[hello]==]", "hello"},
		{"multiline", "[[line1\nline2]]", "line1\nline2"},
		{"skip first newline", "[[\nhello]]", "hello"},
		{"nested brackets", "[[hello [world] ]]", "hello [world] "},
		{"level1", "[=[hello]=]", "hello"},
		{"empty", "[[]]", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := mustTokenize(t, tc.input)
			if tokens[0].Type != token.STRING {
				t.Fatalf("expected STRING, got %v", tokens[0].Type)
			}
			if tokens[0].Literal != tc.expected {
				t.Errorf("value: want %q, got %q", tc.expected, tokens[0].Literal)
			}
		})
	}
}

func TestUnfinishedLongString(t *testing.T) {
	l := New("test", "[[hello", true)
	_, err := l.Next()
	if err == nil {
		t.Fatal("expected error for unfinished long string")
	}
}

func TestShortComments(t *testing.T) {
	tokens := mustTokenize(t, "-- this is a comment\n42")
	if tokens[0].Type != token.INT || tokens[0].IntVal != 42 {
		t.Errorf("expected INT(42) after comment, got %v", tokens[0])
	}
}

func TestLongComments(t *testing.T) {
	tokens := mustTokenize(t, "--[[ this is\na long comment ]]42")
	if tokens[0].Type != token.INT || tokens[0].IntVal != 42 {
		t.Errorf("expected INT(42) after long comment, got %v", tokens[0])
	}
}

func TestLongCommentWithLevel(t *testing.T) {
	tokens := mustTokenize(t, "--[==[ comment ]==]42")
	if tokens[0].Type != token.INT || tokens[0].IntVal != 42 {
		t.Errorf("expected INT(42) after long comment, got %v", tokens[0])
	}
}

func TestCommentAtEOF(t *testing.T) {
	tokens := mustTokenize(t, "-- comment at end")
	if len(tokens) != 1 || tokens[0].Type != token.EOS {
		t.Errorf("expected only EOS after comment, got %v", tokens)
	}
}

func TestBracketNotLongString(t *testing.T) {
	// Single [ is just a bracket
	tokens := mustTokenize(t, "[")
	if tokens[0].Type != token.Type('[') {
		t.Errorf("expected '[', got %v", tokens[0])
	}
}

func TestLineNumbers(t *testing.T) {
	input := "a\nb\n\nc"
	tokens := mustTokenize(t, input)
	// a at line 1, b at line 2, c at line 4
	expected := []int{1, 2, 4}
	for i, line := range expected {
		if tokens[i].Pos.Line != line {
			t.Errorf("token %d (%v): want line %d, got %d",
				i, tokens[i], line, tokens[i].Pos.Line)
		}
	}
}

func TestCompoundExpression(t *testing.T) {
	input := "local x = 42 + y"
	tokens := mustTokenize(t, input)
	expected := []token.Type{
		token.LOCAL, token.NAME, token.Type('='),
		token.INT, token.Type('+'), token.NAME, token.EOS,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, typ := range expected {
		if tokens[i].Type != typ {
			t.Errorf("token %d: want %v, got %v", i, typ, tokens[i].Type)
		}
	}
}

func TestFunctionDecl(t *testing.T) {
	input := `function foo(a, b)
  return a + b
end`
	tokens := mustTokenize(t, input)
	expected := []token.Type{
		token.FUNCTION, token.NAME,
		token.Type('('), token.NAME, token.Type(','), token.NAME, token.Type(')'),
		token.RETURN, token.NAME, token.Type('+'), token.NAME,
		token.END, token.EOS,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, typ := range expected {
		if tokens[i].Type != typ {
			t.Errorf("token %d: want %v, got %v", i, typ, tokens[i].Type)
		}
	}
}

func TestTableConstructor(t *testing.T) {
	input := `{1, "two", [3] = true}`
	tokens := mustTokenize(t, input)
	expected := []token.Type{
		token.Type('{'),
		token.INT, token.Type(','),
		token.STRING, token.Type(','),
		token.Type('['), token.INT, token.Type(']'), token.Type('='), token.TRUE,
		token.Type('}'),
		token.EOS,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, typ := range expected {
		if tokens[i].Type != typ {
			t.Errorf("token %d: want %v, got %v", i, typ, tokens[i].Type)
		}
	}
}

func TestGotoLabel(t *testing.T) {
	input := "::label::"
	tokens := mustTokenize(t, input)
	expected := []token.Type{token.DBCOLON, token.NAME, token.DBCOLON, token.EOS}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, typ := range expected {
		if tokens[i].Type != typ {
			t.Errorf("token %d: want %v, got %v", i, typ, tokens[i].Type)
		}
	}
}

func TestShebang(t *testing.T) {
	// Lua supports #! on the first line as a comment
	input := "#!/usr/bin/lua\nlocal x = 1"
	// Our lexer should handle # as a single char token.
	// Actually Lua's lexer skips the first line if it starts with #.
	// For now, # will be tokenized as a char. The parser can handle shebang.
	tokens := mustTokenize(t, input)
	// Just verify it doesn't error
	if tokens[len(tokens)-1].Type != token.EOS {
		t.Fatal("expected EOS at end")
	}
}

func TestDecimalEscapeTooLarge(t *testing.T) {
	l := New("test", `"\256"`, true)
	_, err := l.Next()
	if err == nil {
		t.Fatal("expected error for decimal escape too large")
	}
}

func TestInvalidEscape(t *testing.T) {
	l := New("test", `"\q"`, true)
	_, err := l.Next()
	if err == nil {
		t.Fatal("expected error for invalid escape")
	}
}

func TestCRLFNewlines(t *testing.T) {
	// \r\n should count as a single newline
	input := "a\r\nb"
	tokens := mustTokenize(t, input)
	if tokens[0].Pos.Line != 1 {
		t.Errorf("first token: want line 1, got %d", tokens[0].Pos.Line)
	}
	if tokens[1].Pos.Line != 2 {
		t.Errorf("second token: want line 2, got %d", tokens[1].Pos.Line)
	}
}

func TestNumberStartingWithDot(t *testing.T) {
	tokens := mustTokenize(t, ".5")
	if tokens[0].Type != token.FLOAT {
		t.Fatalf("expected FLOAT, got %v", tokens[0].Type)
	}
	if tokens[0].FltVal != 0.5 {
		t.Errorf("value: want 0.5, got %g", tokens[0].FltVal)
	}
}

func TestMethodCall(t *testing.T) {
	input := "obj:method()"
	tokens := mustTokenize(t, input)
	expected := []token.Type{
		token.NAME, token.Type(':'), token.NAME,
		token.Type('('), token.Type(')'), token.EOS,
	}
	for i, typ := range expected {
		if tokens[i].Type != typ {
			t.Errorf("token %d: want %v, got %v", i, typ, tokens[i].Type)
		}
	}
}

func TestVarargs(t *testing.T) {
	input := "function f(...) return ... end"
	tokens := mustTokenize(t, input)
	// Check that ... appears twice
	dotsCount := 0
	for _, tok := range tokens {
		if tok.Type == token.DOTS {
			dotsCount++
		}
	}
	if dotsCount != 2 {
		t.Errorf("expected 2 DOTS tokens, got %d", dotsCount)
	}
}

func TestHexIntegerOverflow(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int64
	}{
		{"max_uint64", "0xFFFFFFFFFFFFFFFF", -1},
		{"65bit_zero", "0x10000000000000000", 0},
		{"65bit_minus1", "0x1FFFFFFFFFFFFFFFF", -1},
		{"large_multiword", "0x13121110090807060504030201", 578437695752307201},
		{"allzeros_wrap", "0x100000000000000000000", 0},
		{"single_overflow", "0x10000000000000001", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseInt(tc.input)
			if err != nil {
				t.Fatalf("parseInt(%q) error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("parseInt(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestHexOverflowTokenType(t *testing.T) {
	// Oversized hex literals must tokenize as INT, not FLOAT
	cases := []struct {
		input string
		want  int64
	}{
		{"0x10000000000000000", 0},
		{"0x1FFFFFFFFFFFFFFFF", -1},
		{"0x13121110090807060504030201", 578437695752307201},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			tokens := mustTokenize(t, tc.input)
			if tokens[0].Type != token.INT {
				t.Fatalf("expected INT, got %v", tokens[0].Type)
			}
			if tokens[0].IntVal != tc.want {
				t.Errorf("value: want %d, got %d", tc.want, tokens[0].IntVal)
			}
		})
	}
}

func TestHexOverflowBoundary(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int64
	}{
		{"max_uint64_as_int64", "0xFFFFFFFFFFFFFFFF", -1},
		{"first_overflow", "0x10000000000000000", 0},
		{"leading_zero_64bit", "0x0FFFFFFFFFFFFFFFF", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseInt(tc.input)
			if err != nil {
				t.Fatalf("parseInt(%q) error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("parseInt(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestDecimalOverflowStaysFloat(t *testing.T) {
	cases := []string{
		"99999999999999999999",
		"18446744073709551616", // 2^64
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			tokens := mustTokenize(t, input)
			if tokens[0].Type != token.FLOAT {
				t.Fatalf("expected FLOAT for decimal overflow %q, got %v", input, tokens[0].Type)
			}
		})
	}
}

func TestHexOverflowBitwise(t *testing.T) {
	// Integration: verify bitwise ops work on wrapped hex values via Lua code
	// Tested via parseInt directly since bitwise ops are a VM concern
	cases := []struct {
		name  string
		input string
		mask  int64
		want  int64
	}{
		{"65bit_and_0xFF", "0x10000000000000000", 0xFF, 0},
		{"65bit_plus1_and_0xFF", "0x10000000000000001", 0xFF, 1},
		{"multiword_and_low64", "0x13121110090807060504030201", -1, 578437695752307201},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseInt(tc.input)
			if err != nil {
				t.Fatalf("parseInt(%q) error: %v", tc.input, err)
			}
			result := got & tc.mask
			if result != tc.want {
				t.Errorf("parseInt(%q) & 0x%X = %d, want %d", tc.input, uint64(tc.mask), result, tc.want)
			}
		})
	}
}
