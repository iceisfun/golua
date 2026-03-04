-- Test: math.lua - tonumber with strings
-- From: math.lua
-- What: Tests tonumber with various string formats including valid decimals, invalid strings, signs, and hex integers.

do
  local maxint <const> = math.maxinteger
  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1

  assert(tonumber("0") == 0)
  assert(not tonumber(""))
  assert(not tonumber("  "))
  assert(not tonumber("-"))
  assert(not tonumber("  -0x "))
  assert(not tonumber{})
  assert(tonumber'+0.01' == 1/100 and tonumber'+.01' == 0.01 and
         tonumber'.01' == 0.01    and tonumber'-1.' == -1 and
         tonumber'+1.' == 1)
  assert(not tonumber'+ 0.01' and not tonumber'+.e1' and
         not tonumber'1e'     and not tonumber'1.0e+' and
         not tonumber'.')
  assert(tonumber('-012') == -010-2)
  assert(tonumber('-1.2e2') == - - -120)

  assert(tonumber("0xffffffffffff") == (1 << (4*12)) - 1)
  assert(tonumber("0x"..string.rep("f", (intbits//4))) == -1)
  assert(tonumber("-0x"..string.rep("f", (intbits//4))) == 1)
end
