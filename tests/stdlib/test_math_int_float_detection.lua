-- Test: math.lua - Integer and float bit detection
-- From: math.lua
-- What: Tests detection of integer bit width and floating-point mantissa bits, NaN detection, and math.type.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1
  assert((1 << intbits) == 0)

  assert(minint == 1 << (intbits - 1))
  assert(maxint == minint - 1)

  -- number of bits in the mantissa of a floating-point number
  local floatbits = 24
  do
    local p = 2.0^floatbits
    while p < p + 1.0 do
      p = p * 2.0
      floatbits = floatbits + 1
    end
  end

  local function isNaN (x)
    return (x ~= x)
  end

  assert(isNaN(0/0))
  assert(not isNaN(1/0))


  do
    local x = 2.0^floatbits
    assert(x > x - 1.0 and x == x + 1.0)

    print(string.format("%d-bit integers, %d-bit (mantissa) floats",
                         intbits, floatbits))
  end

  assert(math.type(0) == "integer" and math.type(0.0) == "float"
         and not math.type("10"))
end
