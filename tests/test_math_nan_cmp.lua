-- Test: math.lua - NaN comparisons
-- From: math.lua
-- What: Tests that NaN is not equal to, less than, or less than or equal to any value including itself, minint, and maxint.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local NaN <const> = 0/0
  assert(not (NaN < 0))
  assert(not (NaN > minint))
  assert(not (NaN <= -9))
  assert(not (NaN <= maxint))
  assert(not (NaN < maxint))
  assert(not (minint <= NaN))
  assert(not (minint < NaN))
  assert(not (4 <= NaN))
  assert(not (4 < NaN))
end
