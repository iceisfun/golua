-- Test: math.lua - math.random error cases
-- From: math.lua
-- What: Tests that math.random raises errors for too many arguments and empty intervals.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger
  local random = math.random

  assert(not pcall(random, 1, 2, 3))    -- too many arguments

  -- empty interval
  assert(not pcall(random, minint + 1, minint))
  assert(not pcall(random, maxint, maxint - 1))
  assert(not pcall(random, maxint, minint))
end
