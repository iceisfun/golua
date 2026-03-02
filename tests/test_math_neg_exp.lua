-- Test: math.lua - Negative exponents
-- From: math.lua
-- What: Tests that negative exponents produce correct reciprocal results.

do
  local floatbits = 24
  do
    local p = 2.0^floatbits
    while p < p + 1.0 do
      p = p * 2.0
      floatbits = floatbits + 1
    end
  end

  local function eq (a,b,limit)
    if not limit then
      if floatbits >= 50 then limit = 1E-11
      else limit = 1E-5
      end
    end
    return a == b or math.abs(a-b) <= limit
  end

  do
    assert(2^-3 == 1 / 2^3)
    assert(eq((-3)^-3, 1 / (-3)^3))
    for i = -3, 3 do    -- variables avoid constant folding
        for j = -3, 3 do
          -- domain errors (0^(-n)) are not portable
          if not _port or i ~= 0 or j > 0 then
            assert(eq(i^j, 1 / i^(-j)))
         end
      end
    end
  end
end
