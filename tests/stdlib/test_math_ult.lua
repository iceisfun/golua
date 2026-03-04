-- Test: math.lua - Unsigned comparison (math.ult)
-- From: math.lua
-- What: Tests math.ult for unsigned integer less-than comparisons including negative values and boundary values.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  assert(math.ult(3, 4))
  assert(not math.ult(4, 4))
  assert(math.ult(-2, -1))
  assert(math.ult(2, -1))
  assert(not math.ult(-2, -2))
  assert(math.ult(maxint, minint))
  assert(not math.ult(minint, maxint))
end
