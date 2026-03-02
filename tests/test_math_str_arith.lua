-- Test: math.lua - Implicit string-to-number conversions
-- From: math.lua
-- What: Tests arithmetic operators on string operands with implicit conversion, verifying original strings are unchanged.

do
  local a,b = '10', '20'
  assert(a*b == 200 and a+b == 30 and a-b == -10 and a/b == 0.5 and -b == -20)
  assert(a == '10' and b == '20')
end
