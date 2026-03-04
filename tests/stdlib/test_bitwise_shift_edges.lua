-- Test: bitwise.lua - Shift edge cases
-- From: bitwise.lua
-- What: Tests shifts at and beyond the bit width boundary, negative shifts, and extreme values

do
  local numbits = string.packsize('j') * 8
  assert(-1 >> 1 == (1 << (numbits - 1)) - 1 and 1 << 31 == 0x80000000)
  assert(-1 >> (numbits - 1) == 1)
  assert(-1 >> numbits == 0 and
         -1 >> -numbits == 0 and
         -1 << numbits == 0 and
         -1 << -numbits == 0)
  assert(1 >> math.mininteger == 0)
  assert(1 >> math.maxinteger == 0)
  assert(1 << math.mininteger == 0)
  assert(1 << math.maxinteger == 0)
  assert((2^30 - 1) << 2^30 == 0)
  assert((2^30 - 1) >> 2^30 == 0)
  assert(1 >> -3 == 1 << 3 and 1000 >> 5 == 1000 << -5)
end
