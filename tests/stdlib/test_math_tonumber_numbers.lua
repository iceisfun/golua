-- Test: math.lua - tonumber with numbers
-- From: math.lua
-- What: Tests tonumber on floats, integers, maxint, minint, and infinity.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  assert(tonumber(3.4) == 3.4)
  assert(eqT(tonumber(3), 3))
  assert(eqT(tonumber(maxint), maxint) and eqT(tonumber(minint), minint))
  assert(tonumber(1/0) == 1/0)
end
