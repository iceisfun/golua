package tests

import (
	"strings"
	"testing"
)

func dumpRefLuaChunkHex(t *testing.T, source string) string {
	t.Helper()
	out, err := runRefLua(t, `
local f = `+source+`
local d = string.dump(f, true)
local hex = {}
for i = 1, #d do
  hex[#hex+1] = string.format("%02x", d:byte(i))
end
print(table.concat(hex))
`)
	if err != nil {
		t.Skipf("lua5.4 not available or could not dump chunk: %v", err)
	}
	return strings.TrimSpace(out)
}

func runGoLuaLoadedHexChunk(t *testing.T, hex, prelude string) (string, error) {
	t.Helper()
	code := prelude + `
local hex = "` + hex + `"
local bytes = {}
for i = 1, #hex, 2 do
  bytes[#bytes+1] = string.char(tonumber(hex:sub(i, i + 1), 16))
end
local f = assert(load(table.concat(bytes), "x", "b"))
print(f())
`
	return runGoLua(t, code)
}

// TestImmediateShiftBytecodeParity verifies that precompiled Lua 5.4 bytecode
// using OP_SHLI/OP_SHRI executes with the same results in GoLua.
func TestImmediateShiftBytecodeParity(t *testing.T) {
	hex := dumpRefLuaChunkHex(t, `function() return 3 << 1, 1 << 3, 3 >> 1, 1 >> 3 end`)
	out, err := runGoLuaLoadedHexChunk(t, hex, "")
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "6\t8\t1\t0" {
		t.Fatalf("unexpected output: %q", out)
	}
}

// TestImmediateShiftMetamethodBytecode verifies that precompiled immediate
// shift opcodes preserve Lua 5.4 metamethod fallback behavior.
func TestImmediateShiftMetamethodBytecode(t *testing.T) {
	hex := dumpRefLuaChunkHex(t, `function() return 3 << x, x << 3, 3 >> x, x >> 3 end`)
	prelude := `
x = setmetatable({}, {
  __shl = function(a, b) return "shl:" .. type(a) .. ":" .. type(b) end,
  __shr = function(a, b) return "shr:" .. type(a) .. ":" .. type(b) end,
})
`
	out, err := runGoLuaLoadedHexChunk(t, hex, prelude)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	want := "shl:number:table\tshl:table:number\tshr:number:table\tshr:table:number"
	if out != want {
		t.Fatalf("unexpected output: %q", out)
	}
}

// TestImmediateCompareMetamethodBytecode verifies that precompiled immediate
// comparison opcodes still route through __lt/__le like Lua 5.4.
func TestImmediateCompareMetamethodBytecode(t *testing.T) {
	hex := dumpRefLuaChunkHex(t, `function() return 3 < x, x < 3, 3 <= x, x <= 3 end`)
	prelude := `
x = setmetatable({}, {
  __lt = function(a, b) return true end,
  __le = function(a, b) return true end,
})
`
	out, err := runGoLuaLoadedHexChunk(t, hex, prelude)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "true\ttrue\ttrue\ttrue" {
		t.Fatalf("unexpected output: %q", out)
	}
}
