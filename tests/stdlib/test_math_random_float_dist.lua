-- Test: math.lua - math.random float distribution
-- From: math.lua
-- What: Tests that math.random() (no args) returns floats in [0,1) with correct bit distribution and statistical uniformity.

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

  local random = math.random
  local max = math.max
  local min = math.min

  local function testnear (val, ref, tol)
    return math.abs(val - ref) <= ref * tol
  end

  do   -- test random for floats
    local randbits = math.min(floatbits, 64)   -- at most 64 random bits
    local mult = 2^randbits      -- to make random float into an integral
    local counts = {}    -- counts for bits
    for i = 1, randbits do counts[i] = 0 end
    local up = -math.huge
    local low = math.huge
    local rounds = 100 * randbits   -- 100 times for each bit
    local totalrounds = 0
    ::doagain::   -- will repeat test until we get good statistics
    for i = 0, rounds do
      local t = random()
      assert(0 <= t and t < 1)
      up = max(up, t)
      low = min(low, t)
      assert(t * mult % 1 == 0)    -- no extra bits
      local bit = i % randbits     -- bit to be tested
      if (t * 2^bit) % 1 >= 0.5 then    -- is bit set?
        counts[bit + 1] = counts[bit + 1] + 1   -- increment its count
      end
    end
    totalrounds = totalrounds + rounds
    if not (eq(up, 1, 0.001) and eq(low, 0, 0.001)) then
      goto doagain
    end
    -- all bit counts should be near 50%
    local expected = (totalrounds / randbits / 2)
    for i = 1, randbits do
      if not testnear(counts[i], expected, 0.10) then
        goto doagain
      end
    end
  end
end
