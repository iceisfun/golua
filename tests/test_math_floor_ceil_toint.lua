-- Test: math.lua - Floor, ceil, and tointeger
-- From: math.lua
-- What: Tests math.floor, math.ceil, and math.tointeger with positive/negative floats, integers, minint, maxint, large floats, infinities, NaN, and strings.

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

  do   -- testing floor & ceil
    assert(eqT(math.floor(3.4), 3))
    assert(eqT(math.ceil(3.4), 4))
    assert(eqT(math.floor(-3.4), -4))
    assert(eqT(math.ceil(-3.4), -3))
    assert(eqT(math.floor(maxint), maxint))
    assert(eqT(math.ceil(maxint), maxint))
    assert(eqT(math.floor(minint), minint))
    assert(eqT(math.floor(minint + 0.0), minint))
    assert(eqT(math.ceil(minint), minint))
    assert(eqT(math.ceil(minint + 0.0), minint))
    assert(math.floor(1e50) == 1e50)
    assert(math.ceil(1e50) == 1e50)
    assert(math.floor(-1e50) == -1e50)
    assert(math.ceil(-1e50) == -1e50)
    for _, p in pairs{31,32,63,64} do
      assert(math.floor(2^p) == 2^p)
      assert(math.floor(2^p + 0.5) == 2^p)
      assert(math.ceil(2^p) == 2^p)
      assert(math.ceil(2^p - 0.5) == 2^p)
    end
    checkerror("number expected", math.floor, {})
    checkerror("number expected", math.ceil, print)
    assert(eqT(math.tointeger(minint), minint))
    assert(eqT(math.tointeger(minint .. ""), minint))
    assert(eqT(math.tointeger(maxint), maxint))
    assert(eqT(math.tointeger(maxint .. ""), maxint))
    assert(eqT(math.tointeger(minint + 0.0), minint))
    assert(not math.tointeger(0.0 - minint))
    assert(not math.tointeger(math.pi))
    assert(not math.tointeger(-math.pi))
    assert(math.floor(math.huge) == math.huge)
    assert(math.ceil(math.huge) == math.huge)
    assert(not math.tointeger(math.huge))
    assert(math.floor(-math.huge) == -math.huge)
    assert(math.ceil(-math.huge) == -math.huge)
    assert(not math.tointeger(-math.huge))
    assert(math.tointeger("34.0") == 34)
    assert(not math.tointeger("34.3"))
    assert(not math.tointeger({}))
    assert(not math.tointeger(0/0))    -- NaN
  end
end
