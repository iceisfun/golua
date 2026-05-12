package directives

import (
	"reflect"
	"testing"
)

func mustParse(t *testing.T, src string) *File {
	t.Helper()
	f, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: unexpected error: %v", err)
	}
	if f == nil {
		t.Fatalf("Parse: returned nil *File")
	}
	return f
}

func TestParse_Empty(t *testing.T) {
	f := mustParse(t, "")
	if f.Len() != 0 {
		t.Errorf("Len = %d, want 0", f.Len())
	}
	if got := f.Keys(); got != nil {
		t.Errorf("Keys = %v, want nil", got)
	}
}

func TestParse_SimpleHeader(t *testing.T) {
	src := `-- @tick 30s
-- @scope alias_expander
-- @disabled

return 1
`
	f := mustParse(t, src)

	if v, ok := f.Get("tick"); !ok || v != "30s" {
		t.Errorf("tick: got (%q,%v), want (\"30s\",true)", v, ok)
	}
	if v, ok := f.Get("scope"); !ok || v != "alias_expander" {
		t.Errorf("scope: got (%q,%v), want (\"alias_expander\",true)", v, ok)
	}
	if v, ok := f.Get("disabled"); !ok || v != "" {
		t.Errorf("disabled: got (%q,%v), want (\"\",true)", v, ok)
	}
	if !f.Has("disabled") {
		t.Errorf("Has(disabled) = false, want true")
	}
}

func TestParse_FlagDirective(t *testing.T) {
	for _, src := range []string{
		"-- @disabled\n",
		"-- @disabled",
		"--@disabled\n",
		"--   @disabled   \n",
	} {
		f := mustParse(t, src)
		v, ok := f.Get("disabled")
		if !ok || v != "" {
			t.Errorf("src %q: got (%q,%v), want (\"\",true)", src, v, ok)
		}
	}
}

func TestParse_RepeatedKeys(t *testing.T) {
	src := `-- @import a
-- @import b
-- @import c
return 1
`
	f := mustParse(t, src)

	got := f.Lookup("import")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lookup(import) = %v, want %v", got, want)
	}

	if v, _ := f.Get("import"); v != "c" {
		t.Errorf("Get(import) last-wins = %q, want %q", v, "c")
	}
}

func TestParse_ValuePreservesInternalWhitespace(t *testing.T) {
	src := "-- @desc human-readable text   with   spaces  \n"
	f := mustParse(t, src)
	v, _ := f.Get("desc")
	want := "human-readable text   with   spaces"
	if v != want {
		t.Errorf("desc = %q, want %q", v, want)
	}
}

func TestParse_HeaderTerminates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want bool // whether @scope should be picked up
	}{
		{
			"after code",
			"-- @scope a\nlocal x = 1\n-- @scope b\n",
			true, // b is past code, ignored
		},
		{
			"after long comment",
			"-- @scope a\n--[[ block ]]\n-- @scope b\n",
			true, // long comment terminates header, b ignored
		},
		{
			"blank lines inside header",
			"-- @scope a\n\n\n-- @scope b\nreturn 1\n",
			true, // both inside header
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, tc.src)
			got, _ := f.Get("scope")
			switch tc.name {
			case "after code", "after long comment":
				if got != "a" {
					t.Errorf("scope = %q, want %q (header should have terminated)", got, "a")
				}
			case "blank lines inside header":
				if got != "b" {
					t.Errorf("scope = %q, want %q (last-wins inside header)", got, "b")
				}
			}
		})
	}
}

func TestParse_NonDirectiveCommentDoesNotTerminate(t *testing.T) {
	src := `-- @scope a
-- this is a regular comment
-- another regular comment
-- @tick 5s
return 1
`
	f := mustParse(t, src)
	if v, _ := f.Get("scope"); v != "a" {
		t.Errorf("scope = %q, want \"a\"", v)
	}
	if v, _ := f.Get("tick"); v != "5s" {
		t.Errorf("tick = %q, want \"5s\"", v)
	}
}

func TestParse_Shebang(t *testing.T) {
	src := "#!/usr/bin/env lua\n-- @scope worker\nreturn 1\n"
	f := mustParse(t, src)
	if v, _ := f.Get("scope"); v != "worker" {
		t.Errorf("scope = %q, want \"worker\"", v)
	}
}

func TestParse_ShebangNoNewline(t *testing.T) {
	f := mustParse(t, "#!/usr/bin/env lua")
	if f.Len() != 0 {
		t.Errorf("Len = %d, want 0", f.Len())
	}
}

func TestParse_Malformed(t *testing.T) {
	cases := []string{
		"-- @\n",          // bare @
		"-- @123 value\n", // identifier cannot start with digit
		"-- @-foo bar\n",  // identifier cannot start with dash
	}
	for _, src := range cases {
		f := mustParse(t, src)
		if f.Len() != 0 {
			t.Errorf("src %q: Len = %d, want 0 (malformed should be ignored)", src, f.Len())
		}
	}
}

func TestParse_IdentifierCharset(t *testing.T) {
	src := "-- @my-key_1 value\n"
	f := mustParse(t, src)
	v, ok := f.Get("my-key_1")
	if !ok || v != "value" {
		t.Errorf("my-key_1: got (%q,%v), want (\"value\",true)", v, ok)
	}
}

func TestParse_LongCommentTerminatesHeader(t *testing.T) {
	src := "--[[ banner ]]\n-- @scope a\nreturn 1\n"
	f := mustParse(t, src)
	if f.Has("scope") {
		t.Errorf("scope unexpectedly present after long comment")
	}
}

func TestParse_CRLF(t *testing.T) {
	src := "-- @scope a\r\n-- @tick 5s\r\nreturn 1\r\n"
	f := mustParse(t, src)
	if v, _ := f.Get("scope"); v != "a" {
		t.Errorf("scope = %q, want \"a\"", v)
	}
	if v, _ := f.Get("tick"); v != "5s" {
		t.Errorf("tick = %q, want \"5s\"", v)
	}
}

func TestParse_CaseSensitive(t *testing.T) {
	src := "-- @Scope a\n-- @scope b\n"
	f := mustParse(t, src)
	if v, _ := f.Get("Scope"); v != "a" {
		t.Errorf("Scope = %q, want \"a\"", v)
	}
	if v, _ := f.Get("scope"); v != "b" {
		t.Errorf("scope = %q, want \"b\"", v)
	}
	if f.Has("SCOPE") {
		t.Errorf("Has(SCOPE) true, want false (case-sensitive)")
	}
}

func TestKeysOrder(t *testing.T) {
	src := `-- @b 1
-- @a 2
-- @c 3
-- @a 4
return 1
`
	f := mustParse(t, src)
	got := f.Keys()
	want := []string{"b", "a", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Keys = %v, want %v", got, want)
	}
}

func TestAllIteration(t *testing.T) {
	src := `-- @a 1
-- @b 2
-- @a 3
return 1
`
	f := mustParse(t, src)

	var got [][2]string
	for k, v := range f.All() {
		got = append(got, [2]string{k, v})
	}
	want := [][2]string{{"a", "1"}, {"b", "2"}, {"a", "3"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("All = %v, want %v", got, want)
	}
}

func TestAllEarlyExit(t *testing.T) {
	src := "-- @a 1\n-- @b 2\n-- @c 3\n"
	f := mustParse(t, src)
	count := 0
	for range f.All() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestLookup_Absent(t *testing.T) {
	f := mustParse(t, "-- @a 1\n")
	if got := f.Lookup("missing"); got != nil {
		t.Errorf("Lookup(missing) = %v, want nil", got)
	}
}

func TestGet_Absent(t *testing.T) {
	f := mustParse(t, "")
	v, ok := f.Get("missing")
	if ok || v != "" {
		t.Errorf("Get(missing) = (%q,%v), want (\"\",false)", v, ok)
	}
}

func TestParse_OnlyShebang(t *testing.T) {
	f := mustParse(t, "#!/usr/bin/env lua\n")
	if f.Len() != 0 {
		t.Errorf("Len = %d, want 0", f.Len())
	}
}

func TestParse_NoHeader(t *testing.T) {
	src := `local x = 1
-- @scope ignored
return x
`
	f := mustParse(t, src)
	if f.Len() != 0 {
		t.Errorf("Len = %d, want 0 (no header, all comments past code)", f.Len())
	}
}

func TestParse_LeadingBlankLines(t *testing.T) {
	src := "\n\n\n-- @scope a\nreturn 1\n"
	f := mustParse(t, src)
	if v, _ := f.Get("scope"); v != "a" {
		t.Errorf("scope = %q, want \"a\"", v)
	}
}

func TestParse_NoTrailingNewline(t *testing.T) {
	src := "-- @scope a\n-- @tick 5s"
	f := mustParse(t, src)
	if v, _ := f.Get("tick"); v != "5s" {
		t.Errorf("tick = %q, want \"5s\"", v)
	}
}

// Realistic example from the proposal.
func TestParse_ProposalExample(t *testing.T) {
	src := `-- @tick 30s
-- @scope alias_expander
-- @disabled
-- @import shared/util
-- @import shared/log
-- this comment is ignored, but does not end the header

local function run() return 42 end
return run()
`
	f := mustParse(t, src)

	if v, _ := f.Get("tick"); v != "30s" {
		t.Errorf("tick = %q", v)
	}
	if !f.Has("disabled") {
		t.Errorf("Has(disabled) = false")
	}
	imports := f.Lookup("import")
	want := []string{"shared/util", "shared/log"}
	if !reflect.DeepEqual(imports, want) {
		t.Errorf("imports = %v, want %v", imports, want)
	}
}
