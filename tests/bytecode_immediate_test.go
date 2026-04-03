package tests

import (
	"testing"
)

// TestImmediateShiftBytecodeParity verifies that shift operations produce
// correct results through a dump/load round-trip.
func TestImmediateShiftBytecodeParity(t *testing.T) {
	code := `
local f = function() return 3 << 1, 1 << 3, 3 >> 1, 1 >> 3 end
local d = string.dump(f, true)
local g = assert(load(d, "x", "b"))
print(g())
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "6\t8\t1\t0" {
		t.Fatalf("unexpected output: %q", out)
	}
}

// TestImmediateShiftMetamethodBytecode verifies that shift opcodes preserve
// metamethod fallback behavior through a dump/load round-trip.
func TestImmediateShiftMetamethodBytecode(t *testing.T) {
	code := `
x = setmetatable({}, {
  __shl = function(a, b) return "shl:" .. type(a) .. ":" .. type(b) end,
  __shr = function(a, b) return "shr:" .. type(a) .. ":" .. type(b) end,
})
local f = function() return 3 << x, x << 3, 3 >> x, x >> 3 end
local d = string.dump(f, true)
local g = assert(load(d, "x", "b"))
print(g())
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	want := "shl:number:table\tshl:table:number\tshr:number:table\tshr:table:number"
	if out != want {
		t.Fatalf("unexpected output: %q", out)
	}
}

// TestImmediateCompareMetamethodBytecode verifies that comparison opcodes
// still route through __lt/__le through a dump/load round-trip.
func TestImmediateCompareMetamethodBytecode(t *testing.T) {
	code := `
x = setmetatable({}, {
  __lt = function(a, b) return true end,
  __le = function(a, b) return true end,
})
local f = function() return 3 < x, x < 3, 3 <= x, x <= 3 end
local d = string.dump(f, true)
local g = assert(load(d, "x", "b"))
print(g())
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "true\ttrue\ttrue\ttrue" {
		t.Fatalf("unexpected output: %q", out)
	}
}
