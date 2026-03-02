-- Test: math.lua - Hexadecimal numerals
-- From: math.lua
-- What: Tests hexadecimal integer and float literal parsing, including signed hex, hex with exponent, and potential confusion between decimal 'E' and hex digit 'E'.

do
  assert(0x10 == 16 and 0xfff == 2^12 - 1 and 0XFB == 251)
  assert(0x0p12 == 0 and 0x.0p-3 == 0)
  assert(0xFFFFFFFF == (1 << 32) - 1)
  assert(tonumber('+0x2') == 2)
  assert(tonumber('-0xaA') == -170)
  assert(tonumber('-0xffFFFfff') == -(1 << 32) + 1)

  -- possible confusion with decimal exponent
  assert(0E+1 == 0 and 0xE+1 == 15 and 0xe-1 == 13)
end
