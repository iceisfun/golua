package lexer

import (
	"fmt"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/iceisfun/golua/token"
)

// TestManyConsecutiveCommentsScanIteratively guards the comment scanner against
// consuming a Go frame per comment. Comments produce no token, so a source file
// made of nothing but comments used to grow the stack until a real token
// appeared — a fatal, uncatchable crash on machine-generated input that
// reference Lua lexes in constant stack. The goroutine stack is capped for the
// duration so a regression trips on a short file instead of needing a gigabyte
// of stack to show up.
func TestManyConsecutiveCommentsScanIteratively(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	const n = 200000
	for _, tc := range []struct{ name, comment string }{
		{"short", "--\n"},
		{"long", "--[[c]]"},
		{"nested-level", "--[==[c]==]"},
	} {
		src := strings.Repeat(tc.comment, n) + "return 42"
		tokens, err := New("test", src, false).Tokenize()
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if len(tokens) != 3 || tokens[0].Type != token.RETURN || tokens[1].Type != token.INT || tokens[2].Type != token.EOS {
			t.Fatalf("%s: expected RETURN INT EOS, got %v", tc.name, tokens)
		}
		if tokens[1].IntVal != 42 {
			t.Fatalf("%s: got %d, want 42", tc.name, tokens[1].IntVal)
		}
	}
}

// TestShebangSkipEndsAtLineFeed checks that a leading '#' line is discarded up
// to the first line feed, as the reference loader does. A lone carriage return
// is ordinary text inside that line, not its terminator, so the code after it
// must not be lexed. The line feed that ends the shebang is counted through the
// same path as any other newline, so a carriage return paired with it ("\n\r")
// still ends a single line and everything below keeps its own line number.
func TestShebangSkipEndsAtLineFeed(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		wantLiteral string
		wantLine    int
	}{
		{"lf", "#!/usr/bin/lua\nreturn 1\n", "return", 2},
		{"crlf", "#!/usr/bin/lua\r\nreturn 1\n", "return", 2},
		{"lfcr", "#!/usr/bin/lua\n\rreturn 1\n", "return", 2},
		{"lone-cr", "#!/usr/bin/lua\rlocal skipped = 1\nreturn 2\n", "return", 2},
		{"cr-only-file", "#!/usr/bin/lua\rlocal skipped = 1\r", "", 0},
	} {
		tokens, err := New("test", tc.input, true).Tokenize()
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if tc.wantLiteral == "" {
			if len(tokens) != 1 || tokens[0].Type != token.EOS {
				t.Fatalf("%s: expected only EOS, got %v", tc.name, tokens)
			}
			continue
		}
		if tokens[0].Literal != tc.wantLiteral {
			t.Fatalf("%s: first token %q, want %q", tc.name, tokens[0].Literal, tc.wantLiteral)
		}
		if tokens[0].Pos.Line != tc.wantLine {
			t.Fatalf("%s: first token on line %d, want %d", tc.name, tokens[0].Pos.Line, tc.wantLine)
		}
	}
}

// TestLineTerminatorForms checks the four line terminator forms Lua accepts.
// A carriage return and a line feed each end a line on their own, and a pair of
// different ones ("\r\n" or "\n\r") ends exactly one line between them. The same
// counting has to apply to the terminator of a discarded shebang line, so that
// code below a shebang keeps the line numbers its file has on disk.
func TestLineTerminatorForms(t *testing.T) {
	// Three statements, one per line, joined by the terminator under test.
	const body = "local a%slocal b%sreturn a"

	// wantLines checks that the first token of each of the three statements
	// falls on the expected line.
	wantLines := func(t *testing.T, tokens []token.Token, first int) {
		t.Helper()
		// local a | local b | return a | <eof>
		if len(tokens) != 7 {
			t.Fatalf("got %d tokens, want 7: %v", len(tokens), tokens)
		}
		for i := 0; i < 3; i++ {
			if got, want := tokens[i*2].Pos.Line, first+i; got != want {
				t.Errorf("token %q on line %d, want %d", tokens[i*2].Literal, got, want)
			}
		}
	}

	for _, term := range []struct{ name, sep string }{
		{"lf", "\n"},
		{"cr", "\r"},
		{"crlf", "\r\n"},
		{"lfcr", "\n\r"},
	} {
		src := fmt.Sprintf(body, term.sep, term.sep)

		t.Run("source/"+term.name, func(t *testing.T) {
			tokens, err := New("test", src, false).Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantLines(t, tokens, 1)
		})

		t.Run("shebang/"+term.name, func(t *testing.T) {
			tokens, err := New("test", "#!/usr/bin/lua"+term.sep+src, true).Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if term.sep == "\r" {
				// A lone carriage return does not end the shebang line, and
				// this source has no line feed at all, so every line of it
				// belongs to the discarded first line.
				if len(tokens) != 1 || tokens[0].Type != token.EOS {
					t.Fatalf("expected only EOS, got %v", tokens)
				}
				return
			}
			wantLines(t, tokens, 2)
		})
	}
}
