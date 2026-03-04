-- Test: math.lua - math.modf
-- From: math.lua
-- What: Tests math.modf with positive/negative floats, large numbers, infinities, NaN, integers, and minint.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local function isNaN (x)
    return (x ~= x)
  end

  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  do   -- tests for 'modf'
    local a,b = math.modf(3.5)
    assert(a == 3.0 and b == 0.5)
    a,b = math.modf(-2.5)
    assert(a == -2.0 and b == -0.5)
    a,b = math.modf(-3e23)
    assert(a == -3e23 and b == 0.0)
    a,b = math.modf(3e35)
    assert(a == 3e35 and b == 0.0)
    a,b = math.modf(-1/0)   -- -inf
    assert(a == -1/0 and b == 0.0)
    a,b = math.modf(1/0)   -- inf
    assert(a == 1/0 and b == 0.0)
    a,b = math.modf(0/0)   -- NaN
    assert(isNaN(a) and isNaN(b))
    a,b = math.modf(3)  -- integer argument
    assert(eqT(a, 3) and eqT(b, 0.0))
    a,b = math.modf(minint)
    assert(eqT(a, minint) and eqT(b, 0.0))
  end

  assert(math.huge > 10e30)
  assert(-math.huge < -10e30)
end
