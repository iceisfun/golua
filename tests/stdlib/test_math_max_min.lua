-- Test: math.lua - math.max and math.min
-- From: math.lua
-- What: Tests math.max and math.min with integers, floats, boundary values, and error on missing arguments.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  do    -- testing max/min
    checkerror("value expected", math.max)
    checkerror("value expected", math.min)
    assert(eqT(math.max(3), 3))
    assert(eqT(math.max(3, 5, 9, 1), 9))
    assert(math.max(maxint, 10e60) == 10e60)
    assert(eqT(math.max(minint, minint + 1), minint + 1))
    assert(eqT(math.min(3), 3))
    assert(eqT(math.min(3, 5, 9, 1), 1))
    assert(math.min(3.2, 5.9, -9.2, 1.1) == -9.2)
    assert(math.min(1.9, 1.7, 1.72) == 1.7)
    assert(math.min(-10e60, minint) == -10e60)
    assert(eqT(math.min(maxint, maxint - 1), maxint - 1))
    assert(eqT(math.min(maxint - 2, maxint, maxint - 1), maxint - 2))
  end
end
