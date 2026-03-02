-- Test: math.lua - Integer arithmetic
-- From: math.lua
-- What: Tests integer overflow/wrap-around behavior with minint and maxint.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  assert(minint < minint + 1)
  assert(maxint - 1 < maxint)
  assert(0 - minint == minint)
  assert(minint * minint == 0)
  assert(maxint * maxint * maxint == maxint)
end
