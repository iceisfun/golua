-- Test: math.lua - Float to integer overflow errors at runtime
-- From: math.lua
-- What: Tests that converting +inf, -inf, and NaN to integer via bitwise OR raises errors, and tests float-to-int conversion at precision boundaries.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger
  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1

  local floatbits = 24
  do
    local p = 2.0^floatbits
    while p < p + 1.0 do
      p = p * 2.0
      floatbits = floatbits + 1
    end
  end

  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  local msgf2i = "number.* has no integer representation"

  local function f2i (x) return x | x end
  checkerror(msgf2i, f2i, math.huge)     -- +inf
  checkerror(msgf2i, f2i, -math.huge)    -- -inf
  checkerror(msgf2i, f2i, 0/0)           -- NaN

  if floatbits < intbits then
    -- conversion tests when float cannot represent all integers
    assert(maxint + 1.0 == maxint + 0.0)
    assert(minint - 1.0 == minint + 0.0)
    checkerror(msgf2i, f2i, maxint + 0.0)
    assert(f2i(2.0^(intbits - 2)) == 1 << (intbits - 2))
    assert(f2i(-2.0^(intbits - 2)) == -(1 << (intbits - 2)))
    assert((2.0^(floatbits - 1) + 1.0) // 1 == (1 << (floatbits - 1)) + 1)
    -- maximum integer representable as a float
    local mf = maxint - (1 << (floatbits - intbits)) + 1
    assert(f2i(mf + 0.0) == mf)  -- OK up to here
    mf = mf + 1
    assert(f2i(mf + 0.0) ~= mf)   -- no more representable
  else
    -- conversion tests when float can represent all integers
    assert(maxint + 1.0 > maxint)
    assert(minint - 1.0 < minint)
    assert(f2i(maxint + 0.0) == maxint)
    checkerror("no integer rep", f2i, maxint + 1.0)
    checkerror("no integer rep", f2i, minint - 1.0)
  end

  -- 'minint' should be representable as a float no matter the precision
  assert(f2i(minint + 0.0) == minint)
end
