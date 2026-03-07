-- Test: math.lua - math.randomseed return values
-- From: math.lua
-- What: Tests that math.randomseed returns seeds that can be reused to reproduce the same random sequence.

do
  -- testing return of 'randomseed'
  local x, y = math.randomseed()
  local res = math.random(0)
  x, y = math.randomseed(x, y)    -- should repeat the state
  assert(math.random(0) == res)
  math.randomseed(x, y)    -- again should repeat the state
  assert(math.random(0) == res)
  -- keep the random seed for following tests
end
