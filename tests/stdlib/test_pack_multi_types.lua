-- Test: Multiple types in sequence
-- From: tpack.lua
-- What: Tests packing and unpacking multiple different types (byte, short, float, double, number, int) in a single format string.

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack

do
  local x = pack("<b h b f d f n i", 1, 2, 3, 4, 5, 6, 7, 8)
  assert(#x == packsize("<b h b f d f n i"))
  local a, b, c, d, e, f, g, h = unpack("<b h b f d f n i", x)
  assert(a == 1 and b == 2 and c == 3 and d == 4 and e == 5 and f == 6 and
         g == 7 and h == 8)
end
end
