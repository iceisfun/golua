// This file is an external test package (vm_test) so that it can drive the
// standard library, which imports vm and therefore cannot be imported by an
// internal vm test. Strings produced by Lua code (rep, sub, format, gsub,
// char, table.concat, load) are exactly the runtime-built strings whose data
// pointers differ from a compile-time literal's, so they are what makes the
// content-equality guarantee worth testing.
package vm_test

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// COMPILE-TIME NEGATIVE (cannot be asserted by a normal test, because a test
// that fails to compile fails the whole package). Both of these are rejected
// by the compiler, which is the entire point of the [0]func() guard field in
// vm.Value:
//
//	a, b := vm.NewString("abc"), vm.NewString("abc")
//	_ = a == b                 // invalid operation: a == b
//	                           //   (struct containing [0]func() cannot be compared)
//	_ = map[vm.Value]int{}     // invalid map key type vm.Value
//
// TestValueIsNotComparable below asserts the same property at run time via
// reflect, which is the closest a compiling test can get.

// runLua compiles and runs source on a VM with the full standard library,
// returning the chunk's results.
func runLua(t *testing.T, source string) []vm.Value {
	t.Helper()
	block, err := parser.Parse("<comparability>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<comparability>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	stdlib.Open(v)
	results, err := v.Run(proto)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	return results
}

// TestValueIsNotComparable pins the guard field: vm.Value must not be a
// comparable Go type, so that == and map[vm.Value]T are compile errors rather
// than data-pointer comparisons that answer string equality wrongly.
func TestValueIsNotComparable(t *testing.T) {
	if reflect.TypeOf(vm.Value{}).Comparable() {
		t.Fatal("vm.Value is comparable: Go == on it would compare string data " +
			"pointers, not contents; restore the zero-width non-comparable field")
	}
}

// TestValueSizeAndLayoutUnchanged asserts that making Value non-comparable
// costs nothing. Value was deliberately shrunk to 32 bytes (single-word
// numerics, unboxed strings) and a zero-sized field in *final* position would
// silently add a word of tail padding, so both the total size and every real
// field's offset are compared against a mirror struct with no guard field.
func TestValueSizeAndLayoutUnchanged(t *testing.T) {
	// Mirror of vm.Value's real fields, without the guard field.
	type valueMirror struct {
		typ byte
		n   uint64
		ptr any
	}

	got := reflect.TypeOf(vm.Value{})
	want := reflect.TypeOf(valueMirror{})

	if got.Size() != want.Size() {
		t.Errorf("sizeof(vm.Value) = %d, want %d (the guard field must be first; "+
			"a zero-sized final field adds tail padding)", got.Size(), want.Size())
	}
	if got.Align() != want.Align() {
		t.Errorf("alignof(vm.Value) = %d, want %d", got.Align(), want.Align())
	}

	// On 64-bit platforms the historical, deliberately-tuned size is 32 bytes.
	if unsafe.Sizeof(uintptr(0)) == 8 && unsafe.Sizeof(vm.Value{}) != 32 {
		t.Errorf("sizeof(vm.Value) = %d, want 32", unsafe.Sizeof(vm.Value{}))
	}

	// Compare the real (non-blank) fields positionally: same order, same
	// offsets, same sizes as the guard-free mirror.
	var real []reflect.StructField
	for i := 0; i < got.NumField(); i++ {
		if f := got.Field(i); f.Name != "_" {
			real = append(real, f)
		}
	}
	if len(real) != want.NumField() {
		t.Fatalf("vm.Value has %d non-blank fields, mirror has %d; update the mirror",
			len(real), want.NumField())
	}
	for i, f := range real {
		w := want.Field(i)
		if f.Offset != w.Offset || f.Type.Size() != w.Type.Size() {
			t.Errorf("field %d (%s): offset=%d size=%d, want offset=%d size=%d",
				i, f.Name, f.Offset, f.Type.Size(), w.Offset, w.Type.Size())
		}
	}
}

// stringData returns the address of a string Value's byte data, which is what
// Go's == would have compared. Used only to prove the equality tests below are
// not vacuous (i.e. the Values really are backed by distinct allocations).
func stringData(v vm.Value) uintptr {
	s := v.AsString()
	if len(s) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.StringData(s)))
}

// checkAllEqual asserts that every Value in vals is a string with content want
// and that all of them are mutually Equal and RawEqual. requireDistinct
// additionally demands that at least two of them are backed by different data
// pointers, which is what proves == would have given the wrong answer.
func checkAllEqual(t *testing.T, group string, vals []vm.Value, want string, requireDistinct bool) {
	t.Helper()
	if len(vals) < 2 {
		t.Fatalf("%s: need at least 2 values, got %d", group, len(vals))
	}
	for i, v := range vals {
		if !v.IsString() {
			t.Fatalf("%s[%d]: got type %s, want string", group, i, v.Type())
		}
		if v.AsString() != want {
			t.Fatalf("%s[%d]: contents %q, want %q", group, i, v.AsString(), want)
		}
	}
	for i := range vals {
		for j := range vals {
			if !vals[i].Equal(vals[j]) {
				t.Errorf("%s: Equal(%d, %d) = false, want true (both %q)", group, i, j, want)
			}
			if !vals[i].RawEqual(vals[j]) {
				t.Errorf("%s: RawEqual(%d, %d) = false, want true (both %q)", group, i, j, want)
			}
		}
	}
	if requireDistinct {
		distinct := false
		for i := range vals {
			if stringData(vals[i]) != stringData(vals[0]) {
				distinct = true
				break
			}
		}
		if !distinct {
			t.Errorf("%s: every value shares one data pointer, so this case would "+
				"pass even under pointer equality; the test proves nothing", group)
		}
	}
}

// TestStringValueEqualityAcrossLuaConstructionPaths builds the same string
// through every string-producing path in the standard library and asserts that
// Equal/RawEqual report content equality regardless of how the bytes were
// produced. These are the cases where Go's == is data-dependent: a literal and
// a runtime-built string hold different data pointers.
func TestStringValueEqualityAcrossLuaConstructionPaths(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		source   string
		distinct bool // expect at least two distinct backing allocations
	}{
		{
			name: "plain",
			want: "abc",
			// gsub returns 2 values, so it is parenthesised to truncate to 1;
			// load's result is called and likewise truncated.
			source: `
				local parts = {"a", "b", "c"}
				local built = ""
				for i = 1, #parts do built = built .. parts[i] end
				return "abc",
				       built,
				       string.rep("abc", 1),
				       string.rep("a", 1) .. string.rep("bc", 1),
				       ("xabcx"):sub(2, 4),
				       string.format("%s%s", "ab", "c"),
				       (("a-b-c"):gsub("%-", "")),
				       string.char(97, 98, 99),
				       table.concat(parts),
				       (load("return 'abc'")())
			`,
			distinct: true,
		},
		{
			name: "empty",
			want: "",
			source: `
				return "",
				       ("abc"):sub(4),
				       ("abc"):sub(2, 1),
				       string.rep("x", 0),
				       string.format("%s", ""),
				       (("abc"):gsub("%a", "")),
				       string.char(),
				       table.concat({}),
				       (load("return ''")())
			`,
			// Empty strings have no data pointer to differ on.
			distinct: false,
		},
		{
			name: "embedded NUL",
			want: "a\x00b",
			source: `
				local nul = string.char(0)
				local parts = {"a", nul, "b"}
				local built = ""
				for i = 1, #parts do built = built .. parts[i] end
				return "a\0b",
				       built,
				       string.rep("a\0b", 1),
				       ("xa\0bx"):sub(2, 4),
				       string.format("%s%s%s", "a", nul, "b"),
				       (("a-\0-b"):gsub("%-", "")),
				       string.char(97, 0, 98),
				       table.concat(parts),
				       (load("return 'a\\0b'")())
			`,
			distinct: true,
		},
		{
			name: "NUL only",
			want: "\x00",
			source: `
				return "\0",
				       string.char(0),
				       string.rep("\0", 1),
				       ("x\0x"):sub(2, 2),
				       string.format("%s", string.char(0)),
				       (("-\0-"):gsub("%-", "")),
				       table.concat({string.char(0)}),
				       (load("return '\\0'")())
			`,
			distinct: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkAllEqual(t, tc.name, runLua(t, tc.source), tc.want, tc.distinct)
		})
	}
}

// TestStringValueEqualityAcrossGoConstructionPaths covers the same property on
// the Go side of the embedding API, where the hazard originally surfaced: a
// Value built from a compile-time literal and one built from a runtime-assembled
// string are Equal even though their data pointers differ.
func TestStringValueEqualityAcrossGoConstructionPaths(t *testing.T) {
	lit := vm.NewString("abc")
	joined := vm.NewString(strings.Join([]string{"a", "b", "c"}, ""))
	repeated := vm.NewString(strings.Repeat("abc", 1))
	sliced := vm.NewString(("xabcx")[1:4])
	fromBytes := vm.NewString(string([]byte{'a', 'b', 'c'}))

	vals := []vm.Value{lit, joined, repeated, sliced, fromBytes}
	checkAllEqual(t, "go paths", vals, "abc", true)

	// Empty strings: NewString("") stores a nil data pointer, so this is the
	// one shape where all constructions coincide; assert equality anyway.
	empties := []vm.Value{
		vm.NewString(""),
		vm.NewString(strings.Repeat("x", 0)),
		vm.NewString(strings.Join(nil, "")),
		vm.NewString(string([]byte{})),
	}
	checkAllEqual(t, "go empty", empties, "", false)

	// Embedded NUL bytes must not truncate the comparison.
	nulA := vm.NewString("a\x00b")
	nulB := vm.NewString(strings.Join([]string{"a", "\x00", "b"}, ""))
	checkAllEqual(t, "go nul", []vm.Value{nulA, nulB}, "a\x00b", true)

	// Differing contents must still compare unequal, including strings that
	// share a prefix up to an embedded NUL.
	if nulA.Equal(vm.NewString("a\x00c")) {
		t.Error(`Equal("a\0b", "a\0c") = true, want false`)
	}
	if lit.Equal(vm.NewString("abcd")) {
		t.Error(`Equal("abc", "abcd") = true, want false`)
	}
}

// TestValueMapKeyMigration documents the replacement for map[vm.Value]T, which
// no longer compiles. Keying on the extracted Go value restores the content
// semantics a map key needs: the pointer-keyed map used to hold two entries for
// one logical key.
func TestValueMapKeyMigration(t *testing.T) {
	lit := vm.NewString("abc")
	built := vm.NewString(strings.Join([]string{"a", "b", "c"}, ""))

	// Migration: map[vm.Value]int -> map[string]int keyed on AsString().
	m := map[string]int{}
	m[lit.AsString()] = 1
	m[built.AsString()] = 2

	if len(m) != 1 {
		t.Fatalf("map has %d entries, want 1 (one logical key)", len(m))
	}
	if m["abc"] != 2 {
		t.Errorf(`m["abc"] = %d, want 2`, m["abc"])
	}
}
