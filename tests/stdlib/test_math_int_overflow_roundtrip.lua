-- Test: math.lua - Literal integer overflow (tonumber/tostring round-trip)
-- From: math.lua
-- What: Tests that integer literals at the boundary overflow to floats, and that tonumber/tostring round-trips are correct for maxint/minint.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  do
    -- no overflows
    assert(eqT(tonumber(tostring(maxint)), maxint))
    assert(eqT(tonumber(tostring(minint)), minint))

    -- add 1 to last digit as a string (it cannot be 9...)
    local function incd (n)
      local s = string.format("%d", n)
      s = string.gsub(s, "%d$", function (d)
            assert(d ~= '9')
            return string.char(string.byte(d) + 1)
          end)
      return s
    end

    -- 'tonumber' with overflow by 1
    assert(eqT(tonumber(incd(maxint)), maxint + 1.0))
    assert(eqT(tonumber(incd(minint)), minint - 1.0))

    -- large numbers
    assert(eqT(tonumber("1"..string.rep("0", 30)), 1e30))
    assert(eqT(tonumber("-1"..string.rep("0", 30)), -1e30))

    -- hexa format still wraps around
    assert(eqT(tonumber("0x1"..string.rep("0", 30)), 0))

    -- lexer in the limits
    assert(minint == load("return " .. minint)())
    assert(eqT(maxint, load("return " .. maxint)()))

    assert(eqT(10000000000000000000000.0, 10000000000000000000000))
    assert(eqT(-10000000000000000000000.0, -10000000000000000000000))
  end
end
