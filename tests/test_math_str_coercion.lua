-- Test: math.lua - Numeric string coercion
-- From: math.lua
-- What: Tests string-to-number coercion with leading/trailing spaces and hex prefix.

do
  assert("2" + 1 == 3)
  assert("2 " + 1 == 3)
  assert(" -2 " + 1 == -1)
  assert(" -0xa " + 1 == -9)
end
