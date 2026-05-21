-- test_fuzz_pack_unsigned_wide_sign_extend:
-- string.pack with an unsigned integer format wider than 8 bytes (I9..I16)
-- must ZERO-extend the surplus high bytes, even when the value's int64 bit
-- pattern is negative. golua used to sign-extend (0xFF) unconditionally.
--
-- BROKEN (before fix): stdlib/string_pack.go packInt's size>8 branch set the
-- extension byte to 0xFF whenever val < 0, ignoring the `signed` flag. So
-- string.pack("I16", -1) produced ff*16 instead of ff*8 00*8.
--
-- Lua 5.5's packint is called with neg=0 for the Kuint (unsigned) case, so
-- unsigned formats never sign-extend. As a knock-on effect, golua's own
-- string.unpack("I16", ...) then rejected the bad ff*16 bytes with
-- "16-byte integer does not fit into Lua Integer".
--
-- Reference (lua5.5.0):
--   string.pack("I16", -1)            -> ff ff ff ff ff ff ff ff 00 00 00 00 00 00 00 00
--   string.unpack("I16", pack("I16",-1)) -> -1
--   (signed i16 still sign-extends: string.pack("i16",-1) -> ff*16)
--
-- Discovered: differential scout 2026-05-20 (string-pack agent).

-- unsigned wide pack must zero-extend the high bytes
local p = string.pack("I16", -1)
assert(#p == 16, "I16 must be 16 bytes, got " .. #p)
for i = 1, 8 do
  assert(string.byte(p, i) == 0xFF, "I16(-1) low byte " .. i .. " must be 0xFF")
end
for i = 9, 16 do
  assert(string.byte(p, i) == 0x00,
    "I16(-1) high byte " .. i .. " must zero-extend (was sign-extended)")
end

-- roundtrip through unpack must succeed and return -1
assert(string.unpack("I16", p) == -1, "unpack I16 of packed -1 must be -1")

-- signed wide pack still sign-extends with 0xFF
local s = string.pack("i16", -1)
for i = 1, 16 do
  assert(string.byte(s, i) == 0xFF, "i16(-1) byte " .. i .. " must be 0xFF")
end
assert(string.unpack("i16", s) == -1, "unpack i16 of packed -1 must be -1")

-- a few more sizes / values, signed and unsigned, big- and little-endian
for _, sz in ipairs{9, 10, 12, 16} do
  for _, val in ipairs{-1, -2, 0, 255, 1 << 40} do
    for _, endian in ipairs{"<", ">"} do
      local up = string.unpack(endian .. "I" .. sz,
                               string.pack(endian .. "I" .. sz, val))
      local sp = string.unpack(endian .. "i" .. sz,
                               string.pack(endian .. "i" .. sz, val))
      assert(up == val, "unsigned roundtrip failed: sz=" .. sz .. " val=" .. val)
      assert(sp == val, "signed roundtrip failed: sz=" .. sz .. " val=" .. val)
    end
  end
end

print("ok")
