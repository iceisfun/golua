-- Test: math.lua - Very small numbers and modulo
-- From: math.lua
-- What: Tests modulo behavior with extremely small floating-point numbers near the underflow boundary.

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

  do   -- very small numbers
    local i, j = 0, 20000
    while i < j do
      local m = (i + j) // 2
      if 10^-m > 0 then
        i = m + 1
      else
        j = m
      end
    end
    -- 'i' is the smallest possible ten-exponent
    local b = 10^-(i - (i // 10))   -- a very small number
    assert(b > 0 and b * b == 0)
    local delta = b / 1000
    assert(eq((2.1 * b) % (2 * b), (0.1 * b), delta))
    assert(eq((-2.1 * b) % (2 * b), (2 * b) - (0.1 * b), delta))
    assert(eq((2.1 * b) % (-2 * b), (0.1 * b) - (2 * b), delta))
    assert(eq((-2.1 * b) % (-2 * b), (-0.1 * b), delta))
  end
end
