-- Test: math.lua - Decimal float literals and string conversions
-- From: math.lua
-- What: Tests decimal float literal equivalences, string-to-number for scientific notation, and float ordering.

do
  assert(1.1 == 1.+.1)
  assert(100.0 == 1E2 and .01 == 1e-2)
  assert(1111111111 - 1111111110 == 1000.00e-03)
  assert(1.1 == '1.'+'.1')
  assert(tonumber'1111111111' - tonumber'1111111110' ==
         tonumber"  +0.001e+3 \n\t")

  assert(0.1e-30 > 0.9E-31 and 0.9E30 < 0.1e31)

  assert(0.123456 > 0.123455)

  assert(tonumber('+1.23E18') == 1.23*10.0^18)
end
