package compiler_test

// Execution compatibility with PUC-Rio Lua 5.5 binary chunks.
//
// A few instruction forms are reachable only from a reference chunk: GoLua's
// own code generator resolves arithmetic metamethods inside the arithmetic
// instruction and never emits the OP_MMBIN* follow-ups, and it emits neither
// shift-immediate opcode. Loading a luac chunk is the only way those reach the
// interpreter, so it is the only way to test them.
//
// Each case here pairs the chunk's behaviour with the same source compiled by
// GoLua: the two must agree, and both must agree with what reference Lua prints
// for that source (recorded in the expectations below). Reference Lua is never
// handed GoLua's bytecode — it ships no verifier.

import (
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

// runBothCompilers runs src twice — once compiled by GoLua, once loaded from
// luac5.5.0's chunk for the same source — and checks that both print want.
func runBothCompilers(t *testing.T, src, want string) {
	t.Helper()
	if got := luaOutput(t, src); got != want {
		t.Errorf("GoLua-compiled: got %q, want %q", got, want)
	}
	chunk, _ := luacDump(t, src, false)
	got := luaOutput(t, "local data = ...\nassert(load(data))()\n", vm.NewString(string(chunk)))
	if got != want {
		t.Errorf("%s chunk: got %q, want %q", luacBin, got, want)
	}
}

// An OP_MMBIN* names the left operand in its A field, not the destination; the
// destination belongs to the arithmetic instruction it follows. Writing the
// metamethod's result to A instead overwrites a live operand and leaves the
// real destination stale.
func TestRefChunkArithMetamethodResultDestination(t *testing.T) {
	requireLuac(t)
	runBothCompilers(t, `
local M = setmetatable({}, {__shl = function() return 42 end,
                            __shr = function() return 43 end})
local keep = "kept"
local r = M << 1
local s = M >> 1
print(r, s, type(M), keep)
`, "42\t43\ttable\tkept")
}

// A commutative operator with a constant on the left is compiled with the
// operands commuted so the *K opcode can be used, and the swap is recorded in
// the k bit of the OP_MMBINK that follows. A metamethod has to see them the way
// the source wrote them.
func TestRefChunkCommutedConstantMetamethodOperands(t *testing.T) {
	requireLuac(t)
	runBothCompilers(t, `
local function tag(name)
  return function(a, b) return name .. "(" .. type(a) .. "," .. type(b) .. ")" end
end
local M = setmetatable({}, {
  __add = tag("add"), __mul = tag("mul"),
  __band = tag("band"), __bor = tag("bor"), __bxor = tag("bxor"),
  __sub = tag("sub"), __div = tag("div"),
})
print(1.5 + M, M + 1.5)
print(2 * M, M * 2)
print(7 & M, M & 7)
print(7 | M, M | 7)
print(7 ~ M, M ~ 7)
print(3 - M, M - 3)
print(3 / M, M / 3)
`,
		"add(number,table)\tadd(table,number)\n"+
			"mul(number,table)\tmul(table,number)\n"+
			"band(number,table)\tband(table,number)\n"+
			"bor(number,table)\tbor(table,number)\n"+
			"bxor(number,table)\tbxor(table,number)\n"+
			"sub(number,table)\tsub(table,number)\n"+
			"div(number,table)\tdiv(table,number)")
}

// The shift-immediate opcodes take one operand from a register, so one of the
// two has a run-time shift count: a negative one shifts the other way and a
// count of 64 or more clears the value.
func TestRefChunkShiftImmediateCounts(t *testing.T) {
	requireLuac(t)
	runBothCompilers(t, `
-- x and y are registers, so the count that lands in a register is not folded
-- away and the shift-immediate opcodes are reached with a run-time operand.
local x = -1
local y = 20
print(5 << x, 5 >> x, 255 << -2, 1 >> x, (-1) >> 1)
print(y >> 2, y << 2, y >> -2, y << -2, y >> 64, y << 64, y >> -64, y << -64)

local counts = {-65, -64, -63, -2, -1, 0, 1, 2, 63, 64, 65}
local out = {}
for _, c in ipairs(counts) do
  out[#out + 1] = tostring(5 << c) .. "/" .. tostring(5 >> c)
end
print(table.concat(out, " "))
`,
		"2\t10\t63\t2\t9223372036854775807\n"+
			"5\t80\t80\t5\t0\t0\t0\t0\n"+
			"0/0 0/0 0/-9223372036854775808 1/20 2/10 5/5 10/2 20/1 "+
			"-9223372036854775808/0 0/0 0/0")
}

// An order comparison against a small numeric constant uses the immediate form,
// and the instruction records whether the source wrote that constant as a
// float. An __lt or __le metamethod must see the subtype the source used.
func TestRefChunkOrderImmediateKeepsConstantSubtype(t *testing.T) {
	requireLuac(t)
	runBothCompilers(t, `
local cap
local function f(name)
  return function(x, y) cap = {name, x, y}; return true end
end
local a = setmetatable({}, {__lt = f("lt"), __le = f("le")})
local function show(v)
  if v == a then return "a" end
  return math.type(v) .. ":" .. tostring(v)
end
local function line(label) return label .. " " .. cap[1] .. " " .. show(cap[2]) .. " " .. show(cap[3]) end
local out = {}
local _ = 5.0 > a   ; out[#out + 1] = line("5.0>a")
local _ = a <= -10.0; out[#out + 1] = line("a<=-10.0")
local _ = a < -10   ; out[#out + 1] = line("a<-10")
local _ = 10 >= a   ; out[#out + 1] = line("10>=a")
print(table.concat(out, "\n"))
`,
		"5.0>a lt a float:5.0\n"+
			"a<=-10.0 le a float:-10.0\n"+
			"a<-10 lt a integer:-10\n"+
			"10>=a le a integer:10")
}

// A metamethod still fires when a shift-immediate's register operand is not a
// number, and its result goes to the shift's destination.
func TestRefChunkShiftImmediateMetamethod(t *testing.T) {
	requireLuac(t)
	runBothCompilers(t, `
local M = setmetatable({}, {
  __shl = function(a, b) return "shl(" .. type(a) .. "," .. type(b) .. ")" end,
  __shr = function(a, b) return "shr(" .. type(a) .. "," .. type(b) .. ")" end,
})
local keep = "kept"
print(M << 3, 3 << M, M >> 3, 3 >> M, keep)
`, "shl(table,number)\tshl(number,table)\tshr(table,number)\tshr(number,table)\tkept")
}
