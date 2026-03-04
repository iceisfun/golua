-- Test: math.lua - Modulo with minint and maxint
-- From: math.lua
-- What: Tests modulo behavior with minint and maxint boundary values including negative divisors.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  assert(eqT(minint % minint, 0))
  assert(eqT(maxint % maxint, 0))
  assert((minint + 1) % minint == minint + 1)
  assert((maxint - 1) % maxint == maxint - 1)
  assert(minint % maxint == maxint - 1)

  assert(minint % -1 == 0)
  assert(minint % -2 == 0)
  assert(maxint % -2 == -1)
end
