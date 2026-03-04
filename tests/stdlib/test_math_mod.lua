-- Test: math.lua - Mod operator
-- From: math.lua
-- What: Tests modulo operator with integer and float operands, negative values, and math.pi precision.

do
  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  assert(eqT(-4 % 3, 2))
  assert(eqT(4 % -3, -2))
  assert(eqT(-4.0 % 3, 2.0))
  assert(eqT(4 % -3.0, -2.0))
  assert(eqT(4 % -5, -1))
  assert(eqT(4 % -5.0, -1.0))
  assert(eqT(4 % 5, 4))
  assert(eqT(4 % 5.0, 4.0))
  assert(eqT(-4 % -5, -4))
  assert(eqT(-4 % -5.0, -4.0))
  assert(eqT(-4 % 5, 1))
  assert(eqT(-4 % 5.0, 1.0))
  assert(eqT(4.25 % 4, 0.25))
  assert(eqT(10.0 % 2, 0.0))
  assert(eqT(-10.0 % 2, 0.0))
  assert(eqT(-10.0 % -2, 0.0))
  assert(math.pi - math.pi % 1 == 3)
  assert(math.pi - math.pi % 0.001 == 3.141)
end
