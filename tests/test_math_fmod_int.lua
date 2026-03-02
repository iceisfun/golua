-- Test: math.lua - math.fmod for integers
-- From: math.lua
-- What: Tests math.fmod with integer and float operands, verifying type preservation and consistency with modulo operator for same-sign values.

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

  for i = -6, 6 do
    for j = -6, 6 do
      if j ~= 0 then
        local mi = math.fmod(i, j)
        local mf = math.fmod(i + 0.0, j)
        assert(mi == mf)
        assert(math.type(mi) == 'integer' and math.type(mf) == 'float')
        if (i >= 0 and j >= 0) or (i <= 0 and j <= 0) or mi == 0 then
          assert(eqT(mi, i % j))
        end
      end
    end
  end
  assert(eqT(math.fmod(minint, minint), 0))
  assert(eqT(math.fmod(maxint, maxint), 0))
  assert(eqT(math.fmod(minint + 1, minint), minint + 1))
  assert(eqT(math.fmod(maxint - 1, maxint), maxint - 1))

  checkerror("zero", math.fmod, 3, 0)
end
