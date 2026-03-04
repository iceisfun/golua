-- Test: math.lua - tonumber with base
-- From: math.lua
-- What: Tests tonumber with explicit bases from 2 to 36, including signs, mixed case, and large values.

do
  local maxint <const> = math.maxinteger
  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1

  assert(tonumber('  001010  ', 2) == 10)
  assert(tonumber('  001010  ', 10) == 001010)
  assert(tonumber('  -1010  ', 2) == -10)
  assert(tonumber('10', 36) == 36)
  assert(tonumber('  -10  ', 36) == -36)
  assert(tonumber('  +1Z  ', 36) == 36 + 35)
  assert(tonumber('  -1z  ', 36) == -36 + -35)
  assert(tonumber('-fFfa', 16) == -(10+(16*(15+(16*(15+(16*15)))))))
  assert(tonumber(string.rep('1', (intbits - 2)), 2) + 1 == 2^(intbits - 2))
  assert(tonumber('ffffFFFF', 16)+1 == (1 << 32))
  assert(tonumber('0ffffFFFF', 16)+1 == (1 << 32))
  assert(tonumber('-0ffffffFFFF', 16) - 1 == -(1 << 40))
  for i = 2,36 do
    local i2 = i * i
    local i10 = i2 * i2 * i2 * i2 * i2      -- i^10
    assert(tonumber('\t10000000000\t', i) == i10)
  end
end
