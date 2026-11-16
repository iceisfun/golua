-- Test: numeric strings coerced by string.pack and numeric for loops follow
-- the Lua numeral grammar (integer first, hex wraps, no sign after 0x).
local pack, unpack = string.pack, string.unpack

-- "-0" is the integer 0, so it packs as +0.0
assert(pack("<d", "-0") == pack("<d", 0.0))
assert(pack("<f", "-0") == pack("<f", 0.0))
assert(1 / unpack("<n", pack("<n", "-0")) == math.huge)

-- hex integers wrap modulo 2^64 before becoming floats
assert(unpack("<d", pack("<d", "0xffffffffffffffffff")) == -1.0)
assert(unpack("<d", pack("<d", "0x8000000000000000")) == -2.0^63)

-- a sign after the hex prefix is malformed
for _, s in ipairs{"0x-3", "0x+2", "-0x-3", "0x-1p4"} do
  assert(tonumber(s) == nil, s)
  assert(not pcall(pack, "<j", s), s)
  assert(not pcall(pack, "<d", s), s)
  local ok, err = pcall(function() for i = 1.5, s do end end)
  assert(not ok and err:find("bad 'for' limit", 1, true), s)
  ok, err = pcall(function() for i = s, 3.5 do end end)
  assert(not ok and err:find("bad 'for' initial value", 1, true), s)
  ok, err = pcall(function() for i = 1.5, 3, s do end end)
  assert(not ok and err:find("bad 'for' step", 1, true), s)
end

-- well-formed hex strings still work
assert(unpack("<j", pack("<j", "0x10")) == 16)
assert(unpack("<d", pack("<d", "-0x10")) == -16.0)
assert(unpack("<d", pack("<d", " 0x1p4 ")) == 16.0)
print("PASS")
