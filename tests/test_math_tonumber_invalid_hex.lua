-- Test: math.lua - tonumber invalid hexadecimal formats
-- From: math.lua
-- What: Tests that tonumber returns nil for malformed hex literals including missing digits, double dots, missing exponents.

do
  assert(not tonumber('0x'))
  assert(not tonumber('x'))
  assert(not tonumber('x3'))
  assert(not tonumber('0x3.3.3'))   -- two decimal points
  assert(not tonumber('00x2'))
  assert(not tonumber('0x 2'))
  assert(not tonumber('0 x2'))
  assert(not tonumber('23x'))
  assert(not tonumber('- 0xaa'))
  assert(not tonumber('-0xaaP '))   -- no exponent
  assert(not tonumber('0x0.51p'))
  assert(not tonumber('0x5p+-2'))
end
