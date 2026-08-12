package stdlib

import (
	"runtime"
	"strings"
	"testing"

	"github.com/iceisfun/golua/vm"
)

// capBuilder must refuse an oversized result BEFORE the bytes are copied
// anywhere: a Go runtime OOM is a fatal error that pcall/recover cannot catch,
// so a check that runs after the allocation is no protection at all.
func TestCapBuilderRejectsBeforeCopying(t *testing.T) {
	b := capBuilder{limit: 10}
	b.addString("12345")
	b.addChar('6')

	err := recoverPanic(func() { b.addString("789012") })
	if err != "resulting string too large" {
		t.Fatalf("addString over limit: got %q, want %q", err, "resulting string too large")
	}
	if b.Len() != 6 {
		t.Fatalf("rejected write changed the builder: Len = %d, want 6", b.Len())
	}
	if got := b.String(); got != "123456" {
		t.Fatalf("String after rejected write = %q, want %q", got, "123456")
	}

	// Exactly filling the limit is allowed; one byte past it is not.
	b.addString("7890")
	if err := recoverPanic(func() { b.addChar('x') }); err != "resulting string too large" {
		t.Fatalf("addChar at limit: got %q, want %q", err, "resulting string too large")
	}
	if got := b.String(); got != "1234567890" {
		t.Fatalf("String = %q, want %q", got, "1234567890")
	}
}

// Large pieces are held by reference and joined at the end, so the interleaving
// with copied small pieces has to preserve order.
func TestCapBuilderPreservesPieceOrder(t *testing.T) {
	big1 := strings.Repeat("A", largePiece)
	big2 := strings.Repeat("B", largePiece*2)

	b := capBuilder{limit: maxStrResultSize}
	b.addChar('<')
	b.addString("small")
	b.addString(big1)
	b.addString("mid")
	b.addString(big2)
	b.addChar('>')
	b.addString(big1)

	want := "<small" + big1 + "mid" + big2 + ">" + big1
	if got := b.String(); got != want {
		t.Fatalf("String length %d, want %d (content mismatch)", len(got), len(want))
	}
	if b.Len() != len(want) {
		t.Fatalf("Len = %d, want %d", b.Len(), len(want))
	}
}

// string.format used to build into an unbounded buffer: a handful of %s
// conversions over a large string killed the host with a fatal OOM. The cap now
// fires before the bytes are ever accumulated, so the rejection is both
// catchable and cheap — the memory assertion is the actual regression guard.
func TestFormatOversizedResultIsCatchableAndCheap(t *testing.T) {
	m := vm.New()
	Open(m)

	alloc := measureAlloc(t, func() {
		err := runLuaWithVM(t, m, `
local a = string.rep("a", 16 * 1024 * 1024)
local args = {}
for i = 1, 100 do args[i] = a end
local ok, err = pcall(string.format, string.rep("%s", 100), table.unpack(args))
assert(ok == false, "oversized string.format must fail")
assert(string.find(err, "resulting string too large", 1, true), "unexpected error: " .. tostring(err))
`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	// The refused result would have been 1.6GB; only the 16MB argument and a
	// little churn should ever be allocated.
	if alloc > 256<<20 {
		t.Fatalf("format allocated %d bytes building a result it then rejected", alloc)
	}
}

// %q expands its argument, so its size is measured before it is built.
func TestFormatQuoteOversizedIsCatchableAndCheap(t *testing.T) {
	m := vm.New()
	Open(m)

	alloc := measureAlloc(t, func() {
		err := runLuaWithVM(t, m, `
local a = string.rep("a", 16 * 1024 * 1024)
local args = {}
for i = 1, 70 do args[i] = a end
args[71] = "\1\1\1"
-- 70 x 16MB of %s leaves the %q expansion without room; it must be refused
-- rather than escaped first.
local ok, err = pcall(string.format, string.rep("%s", 70) .. "%q", table.unpack(args))
assert(ok == false, "oversized %q must fail")
assert(string.find(err, "resulting string too large", 1, true), "unexpected error: " .. tostring(err))
`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if alloc > 256<<20 {
		t.Fatalf("format allocated %d bytes for a result it then rejected", alloc)
	}
}

// gsub expanded a single replacement into its own unbounded buffer before the
// accumulated-size guard ever ran.
func TestGsubOversizedReplacementIsCatchableAndCheap(t *testing.T) {
	m := vm.New()
	Open(m)

	alloc := measureAlloc(t, func() {
		err := runLuaWithVM(t, m, `
local a = string.rep("a", 16 * 1024 * 1024)
local ok, err = pcall(string.gsub, a, "(.*)", string.rep("%1", 100))
assert(ok == false, "oversized string.gsub must fail")
assert(string.find(err, "resulting string too large", 1, true), "unexpected error: " .. tostring(err))
`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if alloc > 256<<20 {
		t.Fatalf("gsub allocated %d bytes expanding a replacement it then rejected", alloc)
	}
}

// The size %q reserves must be exactly the size it writes, for every byte and
// every following byte (the escape form depends on whether a digit follows).
func TestQuotedByteLenMatchesQuotedOutput(t *testing.T) {
	m := vm.New()
	pair := make([]byte, 2)
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			pair[0], pair[1] = byte(a), byte(b)
			s := string(pair)
			want := 2 // surrounding quotes
			for i := 0; i < len(s); i++ {
				want += quotedByteLen(s, i)
			}
			got := luaFormatValues(m, "%q", []vm.Value{vm.NewString(s)})
			if len(got) != want {
				t.Fatalf("%%q of %q: quotedByteLen sum %d, actual %d (%q)", s, want, len(got), got)
			}
		}
	}
}

// Ordinary formatting and substitution must be byte-identical to before the cap
// was added (values checked against lua5.5.0).
func TestFormatAndGsubUnaffectedByCap(t *testing.T) {
	m := vm.New()
	Open(m)

	err := runLuaWithVM(t, m, `
assert(string.format("%s-%d-%5.2f", "x", 42, 3.14159) == "x-42- 3.14")
assert(string.format("%q", "a\nb\"c\\d\1e\0022f") == '"a\\\nb\\"c\\\\d\\1e\\0022f"')
assert(string.format("%10s|%-10s|", "ab", "cd") == "        ab|cd        |")
assert(string.format("%c%c%c", 65, 66, 67) == "ABC")
assert(string.format("[%5.3s]", "abcdef") == "[  abc]")
assert(("hello world"):gsub("o", "0") == "hell0 w0rld")
local r, n = ("hello"):gsub("(l)(l)", "%2%1")
assert(r == "hello" and n == 1)
local r2, n2 = ("abc"):gsub("", "-")
assert(r2 == "-a-b-c-" and n2 == 4)
assert(("aaa"):gsub("a", "%0%0", 2) == "aaaaa")
assert(("x=1"):gsub("(%w+)=(%w+)", "%2=%1") == "1=x")
local big = string.rep("ab", 5000)
local rep = big:gsub("ab", "cd")
assert(rep == string.rep("cd", 5000))
assert(#string.format("%s%s", big, big) == 2 * #big)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// recoverPanic runs fn and returns the recovered panic value as a string.
func recoverPanic(fn func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case *vm.LuaError:
				msg = x.Value.String()
			case error:
				msg = x.Error()
			case string:
				msg = x
			}
		}
	}()
	fn()
	return ""
}

// measureAlloc reports how many bytes fn allocated in total.
func measureAlloc(t *testing.T, fn func()) uint64 {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
