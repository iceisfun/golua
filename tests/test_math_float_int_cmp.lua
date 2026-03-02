-- Test: math.lua - Float and integer comparison border cases
-- From: math.lua
-- What: Tests comparison operators between floats and integers at precision boundaries, including floatbits vs intbits divergence.

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

  if floatbits < intbits then
    assert(2.0^floatbits == (1 << floatbits))
    assert(2.0^floatbits - 1.0 == (1 << floatbits) - 1.0)
    assert(2.0^floatbits - 1.0 ~= (1 << floatbits))
    -- float is rounded, int is not
    assert(2.0^floatbits + 1.0 ~= (1 << floatbits) + 1)
  else   -- floats can express all integers with full accuracy
    assert(maxint == maxint + 0.0)
    assert(maxint - 1 == maxint - 1.0)
    assert(minint + 1 == minint + 1.0)
    assert(maxint ~= maxint - 1.0)
  end
  assert(maxint + 0.0 == 2.0^(intbits - 1) - 1.0)
  assert(minint + 0.0 == minint)
  assert(minint + 0.0 == -2.0^(intbits - 1))
end
