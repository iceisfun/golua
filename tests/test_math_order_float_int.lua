-- Test: math.lua - Order between floats and integers
-- From: math.lua
-- What: Tests less-than and less-equal ordering between integer and float values near the limits of representation.

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

  assert(1 < 1.1); assert(not (1 < 0.9))
  assert(1 <= 1.1); assert(not (1 <= 0.9))
  assert(-1 < -0.9); assert(not (-1 < -1.1))
  assert(1 <= 1.1); assert(not (-1 <= -1.1))
  assert(-1 < -0.9); assert(not (-1 < -1.1))
  assert(-1 <= -0.9); assert(not (-1 <= -1.1))
  assert(minint <= minint + 0.0)
  assert(minint + 0.0 <= minint)
  assert(not (minint < minint + 0.0))
  assert(not (minint + 0.0 < minint))
  assert(maxint < minint * -1.0)
  assert(maxint <= minint * -1.0)

  do
    local fmaxi1 = 2^(intbits - 1)
    assert(maxint < fmaxi1)
    assert(maxint <= fmaxi1)
    assert(not (fmaxi1 <= maxint))
    assert(minint <= -2^(intbits - 1))
    assert(-2^(intbits - 1) <= minint)
  end

  if floatbits < intbits then
    print("testing order (floats cannot represent all integers)")
    local fmax = 2^floatbits
    local ifmax = fmax | 0
    assert(fmax < ifmax + 1)
    assert(fmax - 1 < ifmax)
    assert(-(fmax - 1) > -ifmax)
    assert(not (fmax <= ifmax - 1))
    assert(-fmax > -(ifmax + 1))
    assert(not (-fmax >= -(ifmax - 1)))

    assert(fmax/2 - 0.5 < ifmax//2)
    assert(-(fmax/2 - 0.5) > -ifmax//2)

    assert(maxint < 2^intbits)
    assert(minint > -2^intbits)
    assert(maxint <= 2^intbits)
    assert(minint >= -2^intbits)
  else
    print("testing order (floats can represent all integers)")
    assert(maxint < maxint + 1.0)
    assert(maxint < maxint + 0.5)
    assert(maxint - 1.0 < maxint)
    assert(maxint - 0.5 < maxint)
    assert(not (maxint + 0.0 < maxint))
    assert(maxint + 0.0 <= maxint)
    assert(not (maxint < maxint + 0.0))
    assert(maxint + 0.0 <= maxint)
    assert(maxint <= maxint + 0.0)
    assert(not (maxint + 1.0 <= maxint))
    assert(not (maxint + 0.5 <= maxint))
    assert(not (maxint <= maxint - 1.0))
    assert(not (maxint <= maxint - 0.5))

    assert(minint < minint + 1.0)
    assert(minint < minint + 0.5)
    assert(minint <= minint + 0.5)
    assert(minint - 1.0 < minint)
    assert(minint - 1.0 <= minint)
    assert(not (minint + 0.0 < minint))
    assert(not (minint + 0.5 < minint))
    assert(not (minint < minint + 0.0))
    assert(minint + 0.0 <= minint)
    assert(minint <= minint + 0.0)
    assert(not (minint + 1.0 <= minint))
    assert(not (minint + 0.5 <= minint))
    assert(not (minint <= minint - 1.0))
  end
end
