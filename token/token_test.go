package token

import "testing"

func TestLookupIdent(t *testing.T) {
	if LookupIdent("and") != AND {
		t.Error("expected AND for 'and'")
	}
	if LookupIdent("while") != WHILE {
		t.Error("expected WHILE for 'while'")
	}
	if LookupIdent("global") != GLOBAL {
		t.Error("expected GLOBAL for 'global'")
	}
	if LookupIdent("myvar") != NAME {
		t.Error("expected NAME for 'myvar'")
	}
}

func TestIsKeyword(t *testing.T) {
	if !AND.IsKeyword() {
		t.Error("AND should be keyword")
	}
	if !WHILE.IsKeyword() {
		t.Error("WHILE should be keyword")
	}
	if NAME.IsKeyword() {
		t.Error("NAME should not be keyword")
	}
	if EOS.IsKeyword() {
		t.Error("EOS should not be keyword")
	}
}

func TestTypeString(t *testing.T) {
	cases := []struct {
		typ    Type
		expect string
	}{
		{AND, "and"},
		{WHILE, "while"},
		{EQ, "=="},
		{CONCAT, ".."},
		{DOTS, "..."},
		{EOS, "<eof>"},
		{Type('+'), "+"},
		{NAME, "<name>"},
		{INT, "<integer>"},
		{FLOAT, "<number>"},
		{STRING, "<string>"},
	}
	for _, tc := range cases {
		if got := tc.typ.String(); got != tc.expect {
			t.Errorf("Type(%d).String() = %q, want %q", tc.typ, got, tc.expect)
		}
	}
}

func TestPosString(t *testing.T) {
	p := Pos{Source: "test.lua", Line: 10, Column: 5}
	if got := p.String(); got != "test.lua:10:5" {
		t.Errorf("Pos.String() = %q, want %q", got, "test.lua:10:5")
	}
}

func TestTokenString(t *testing.T) {
	tok := Token{Type: NAME, Literal: "foo", Pos: Pos{Source: "test", Line: 1, Column: 1}}
	s := tok.String()
	if s != `<name>("foo")` {
		t.Errorf("Token.String() = %q, want %q", s, `<name>("foo")`)
	}

	tok2 := Token{Type: INT, Literal: "42", IntVal: 42}
	if got := tok2.String(); got != "<integer>(42)" {
		t.Errorf("INT token String() = %q, want %q", got, "<integer>(42)")
	}

	tok3 := Token{Type: EOS}
	if got := tok3.String(); got != "<eof>" {
		t.Errorf("EOS token String() = %q, want %q", got, "<eof>")
	}
}

func TestPosError(t *testing.T) {
	e := &PosError{
		Pos: Pos{Source: "test.lua", Line: 5, Column: 10},
		Msg: "unexpected symbol",
	}
	want := "test.lua:5: unexpected symbol"
	if got := e.Error(); got != want {
		t.Errorf("PosError.Error() = %q, want %q", got, want)
	}

	// Verify it satisfies the error interface.
	var err error = e
	if err.Error() != want {
		t.Errorf("error interface: got %q, want %q", err.Error(), want)
	}
}
