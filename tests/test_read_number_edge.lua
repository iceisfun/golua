-- Test read("n") edge cases

-- Helper: write string to tmpfile, seek to start, return file
local function mkf(s)
  local f = io.tmpfile()
  f:write(s)
  f:seek("set")
  return f
end

-- Test: signed hex numbers
local f = mkf("-0xff")
local n = f:read("n")
assert(n == -255, "T_neg_hex: expected -255, got "..tostring(n))
assert(math.type(n) == "integer", "T_neg_hex: expected integer, got "..math.type(n))

f = mkf("+0xff")
n = f:read("n")
assert(n == 255, "T_pos_hex: expected 255, got "..tostring(n))
assert(math.type(n) == "integer", "T_pos_hex: expected integer, got "..math.type(n))

f = mkf("-0X1A")
n = f:read("n")
assert(n == -26, "T_neg_0X1A: expected -26, got "..tostring(n))

f = mkf("+0X1A")
n = f:read("n")
assert(n == 26, "T_pos_0X1A: expected 26, got "..tostring(n))

-- Test: incomplete exponent is a hard failure (no fallback to shorter number)
f = mkf("1e\n")
n = f:read("n")
assert(n == nil, "T_1e: expected nil, got "..tostring(n))

f = mkf("1E\n")
n = f:read("n")
assert(n == nil, "T_1E: expected nil, got "..tostring(n))

f = mkf("1e+\n")
n = f:read("n")
assert(n == nil, "T_1e+: expected nil, got "..tostring(n))

f = mkf("1e-\n")
n = f:read("n")
assert(n == nil, "T_1e-: expected nil, got "..tostring(n))

f = mkf("0x1p\n")
n = f:read("n")
assert(n == nil, "T_0x1p: expected nil, got "..tostring(n))

f = mkf("0x1P\n")
n = f:read("n")
assert(n == nil, "T_0x1P: expected nil, got "..tostring(n))

f = mkf("0x1p+\n")
n = f:read("n")
assert(n == nil, "T_0x1p+: expected nil, got "..tostring(n))

f = mkf("0x1p-\n")
n = f:read("n")
assert(n == nil, "T_0x1p-: expected nil, got "..tostring(n))

-- Test: failed read("n") leaves position at end of scanned chars (not restored)
-- Lua 5.4: "+\n" -> nil, pos=1 (consumed the +)
f = mkf("+\n")
n = f:read("n")
assert(n == nil, "T_plus_nl: expected nil")
local l = f:read("l")
assert(l == "", "T_plus_nl: expected empty line after +, got ["..tostring(l).."]")

-- "0x\n" -> nil, pos=2 (consumed the 0x)
f = mkf("0x\n")
n = f:read("n")
assert(n == nil, "T_0x_nl: expected nil")
l = f:read("l")
assert(l == "", "T_0x_nl: expected empty line, got ["..tostring(l).."]")

-- "0xGG" -> nil, pos=2 (consumed the 0x)
f = mkf("0xGG")
n = f:read("n")
assert(n == nil, "T_0xGG: expected nil")
l = f:read("l")
assert(l == "GG", "T_0xGG: expected 'GG', got ["..tostring(l).."]")

-- "+abc" -> nil, pos=1 (consumed the +)
f = mkf("+abc")
n = f:read("n")
assert(n == nil, "T_plus_abc: expected nil")
l = f:read("l")
assert(l == "abc", "T_plus_abc: expected 'abc', got ["..tostring(l).."]")

-- "1e+" -> nil, pos=3 (consumed 1e+)
f = mkf("1e+\n")
n = f:read("n")
assert(n == nil, "T_1e+_pos: expected nil")
local pos = f:seek("cur")
assert(pos == 3, "T_1e+_pos: expected pos=3, got "..pos)

-- "0x1p+" -> nil, pos=5 (consumed 0x1p+)
f = mkf("0x1p+\n")
n = f:read("n")
assert(n == nil, "T_0x1p+_pos: expected nil")
pos = f:seek("cur")
assert(pos == 5, "T_0x1p+_pos: expected pos=5, got "..pos)

-- "1e" -> nil, pos=2
f = mkf("1e\n")
n = f:read("n")
assert(n == nil, "T_1e_pos: expected nil")
pos = f:seek("cur")
assert(pos == 2, "T_1e_pos: expected pos=2, got "..pos)

-- Existing behavior that should still work:
-- Complete exponents work fine
f = mkf("1e+2rest")
n = f:read("n")
assert(n == 100.0, "T_1e+2: expected 100.0, got "..tostring(n))
l = f:read("l")
assert(l == "rest", "T_1e+2_rest: expected 'rest', got ["..tostring(l).."]")

-- Normal hex
f = mkf("0xff")
n = f:read("n")
assert(n == 255, "T_0xff: expected 255, got "..tostring(n))

-- Hex float
f = mkf("0x1.8p1")
n = f:read("n")
assert(n == 3.0, "T_hexfloat: expected 3.0, got "..tostring(n))

print("PASS")
