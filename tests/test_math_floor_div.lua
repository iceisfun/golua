-- Test: math.lua - Floor division and conversions
-- From: math.lua
-- What: Tests integer and float floor division, verifying consistency with math.floor(i/j) and type preservation.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger
  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1

  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  for _, i in pairs{-16, -15, -3, -2, -1, 0, 1, 2, 3, 15} do
    for _, j in pairs{-16, -15, -3, -2, -1, 1, 2, 3, 15} do
      for _, ti in pairs{0, 0.0} do     -- try 'i' as integer and as float
        for _, tj in pairs{0, 0.0} do   -- try 'j' as integer and as float
          local x = i + ti
          local y = j + tj
            assert(i//j == math.floor(i/j))
        end
      end
    end
  end

  assert(1//0.0 == 1/0)
  assert(-1 // 0.0 == -1/0)
  assert(eqT(3.5 // 1.5, 2.0))
  assert(eqT(3.5 // -1.5, -3.0))

  do   -- tests for different kinds of opcodes
    local x, y
    x = 1; assert(x // 0.0 == 1/0)
    x = 1.0; assert(x // 0 == 1/0)
    x = 3.5; assert(eqT(x // 1, 3.0))
    assert(eqT(x // -1, -4.0))

    x = 3.5; y = 1.5; assert(eqT(x // y, 2.0))
    x = 3.5; y = -1.5; assert(eqT(x // y, -3.0))
  end

  assert(maxint // maxint == 1)
  assert(maxint // 1 == maxint)
  assert((maxint - 1) // maxint == 0)
  assert(maxint // (maxint - 1) == 1)
  assert(minint // minint == 1)
  assert(minint // minint == 1)
  assert((minint + 1) // minint == 0)
  assert(minint // (minint + 1) == 1)
  assert(minint // 1 == minint)

  assert(minint // -1 == -minint)
  assert(minint // -2 == 2^(intbits - 2))
  assert(maxint // -1 == -maxint)
end
