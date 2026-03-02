-- Test: math.lua - Floating hexas
-- From: math.lua
-- What: Tests hexadecimal floating-point literals with fractional parts and binary exponents.

do
  assert(tonumber('  0x2.5  ') == 0x25/16)
  assert(tonumber('  -0x2.5  ') == -0x25/16)
  assert(tonumber('  +0x0.51p+8  ') == 0x51)
  assert(0x.FfffFFFF == 1 - '0x.00000001')
  assert('0xA.a' + 0 == 10 + 10/16)
  assert(0xa.aP4 == 0XAA)
  assert(0x4P-2 == 1)
  assert(0x1.1 == '0x1.' + '+0x.1')
  assert(0Xabcdef.0 == 0x.ABCDEFp+24)
end
