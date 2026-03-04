-- Test: math.lua - math.random seed and raw value
-- From: math.lua
-- What: Tests the low-level random number generator output after a specific seed, verifying the implementation matches the expected 64-bit state.

do
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

  local function eq (a,b,limit)
    if not limit then
      if floatbits >= 50 then limit = 1E-11
      else limit = 1E-5
      end
    end
    return a == b or math.abs(a-b) <= limit
  end

  local random = math.random

  -- all computations should work with 32-bit integers
  local h <const> = 0x7a7040a5   -- higher half
  local l <const> = 0xa323c9d6   -- lower half

  math.randomseed(1007)
  -- get the low 'intbits' of the 64-bit expected result
  local res = (h << 32 | l) & ~(~0 << intbits)
  assert(random(0) == res)

  math.randomseed(1007, 0)
  -- using higher bits to generate random floats; (the '% 2^32' converts
  -- 32-bit integers to floats as unsigned)
  local res
  if floatbits <= 32 then
    -- get all bits from the higher half
    res = (h >> (32 - floatbits)) % 2^32
  else
    -- get 32 bits from the higher half and the rest from the lower half
    res = (h % 2^32) * 2^(floatbits - 32) + ((l >> (64 - floatbits)) % 2^32)
  end
  local rand = random()
  assert(eq(rand, 0x0.7a7040a5a323c9d6, 2^-floatbits))
  assert(rand * 2^floatbits == res)
end
