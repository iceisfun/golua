package golua_test

import (
	"strings"
	"testing"
)

// string->number coercion. Each snippet prints one line;
// we assert the output contains the expected substring (error messages carry a
// chunk-name prefix, so a Contains check is used throughout).
func TestStringCoercion(t *testing.T) {
	cases := []struct{ src, want string }{
		{`print(pcall(function() return "1_0" + 0 end))`, "attempt to add a 'string' with a 'number'"},
		{`print(tonumber("1_0"))`, "nil"},
		{`print(string.format("%d", "0x10000000000000000"))`, "0"},
		{`print(string.format("%d", "0xff"))`, "255"},
		{`print(tonumber("0x1.921fb54442d18p+1"))`, "3.1415926535898"}, // Lua 5.4 %.14g display
		{`print(("0xa.28p33")+0.0)`, "87241523200.0"},
		{`print(tonumber("0x1.8"))`, "1.5"},
	}
	for _, c := range cases {
		if got := runLuaCapture(t, c.src); !strings.Contains(got, c.want) {
			t.Errorf("%s => got %q want substring %q", c.src, got, c.want)
		}
	}
}

// string.format flag/width edge cases.
func TestStringFormat(t *testing.T) {
	cases := []struct{ src, want string }{
		{`print("["..string.format("%--5.2s","abcdef").."]")`, "[ab   ]"},
		{`print("["..string.format("%--10s","hi").."]")`, "[hi        ]"},
		{`print("["..string.format("%##x",0).."]")`, "[0]"},
		{`print("["..string.format("%##X",0).."]")`, "[0]"},
		{`print("["..string.format("%#x",255).."]")`, "[0xff]"},
		{`print(pcall(string.format, "%9999999999999999999d", 1))`, "invalid conversion specification: '%9999999999999999999d'"},
		{`print(pcall(string.format, "%.9999999999999999999f", 1.5))`, "invalid conversion specification: '%.9999999999999999999f'"},
	}
	for _, c := range cases {
		if got := runLuaCapture(t, c.src); !strings.Contains(got, c.want) {
			t.Errorf("%s => got %q want substring %q", c.src, got, c.want)
		}
	}
}

// string.pack size parsing and unpack alignment bounds.
func TestStringPack(t *testing.T) {
	if got := runLuaCapture(t, `print(pcall(string.packsize, "c9223372036854775807"))`); !strings.Contains(got, "invalid format option '6'") {
		t.Errorf("packsize 19-digit c: got %q", got)
	}
	if got := runLuaCapture(t, `print(pcall(string.unpack, "!xXh", "x"))`); !strings.Contains(got, "data string too short") {
		t.Errorf("unpack X past end: got %q", got)
	}
}

// string.unpack must grow the stack per decoded value; >=252 results used to
// run off the fixed frame and leak a Go index-out-of-range panic to the host.
func TestStringUnpackManyResults(t *testing.T) {
	got := runLuaCapture(t, `
local n = 300
print(select("#", string.unpack(string.rep("B", n), string.rep("\7", n))))
local t = {string.unpack(string.rep("B", n), string.rep("\7", n))}
print(#t, t[1], t[n], t[n+1])`)
	want := "301\n301\t7\t7\t301"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// string.gsub with a number subject must return a string even on the
// no-change fast path (reference coerces the argument slot in place).
func TestStringGsubNumberSubject(t *testing.T) {
	got := runLuaCapture(t, `
print(type(string.gsub(1, "x", "y")))
print(type(string.gsub(123, "%d", "%0")))
print(type(string.gsub(12.5, "9", "X")))
print(string.gsub(123, "9", "X"))`)
	want := "string\nstring\nstring\n123\t0"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
