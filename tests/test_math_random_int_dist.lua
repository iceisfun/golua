-- Test: math.lua - math.random full integer distribution
-- From: math.lua
-- What: Tests that math.random(0) returns full-range integers with uniform bit distribution.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger
  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1

  local random = math.random
  local max = math.max
  local min = math.min

  local function testnear (val, ref, tol)
    return math.abs(val - ref) <= ref * tol
  end

  do   -- test random for full integers
    local up = 0
    local low = 0
    local counts = {}    -- counts for bits
    for i = 1, intbits do counts[i] = 0 end
    local rounds = 100 * intbits   -- 100 times for each bit
    local totalrounds = 0
    ::doagain::   -- will repeat test until we get good statistics
    for i = 0, rounds do
      local t = random(0)
      up = max(up, t)
      low = min(low, t)
      local bit = i % intbits     -- bit to be tested
      -- increment its count if it is set
      counts[bit + 1] = counts[bit + 1] + ((t >> bit) & 1)
    end
    totalrounds = totalrounds + rounds
    local lim = maxint >> 10
    if not (maxint - up < lim and low - minint < lim) then
      goto doagain
    end
    -- all bit counts should be near 50%
    local expected = (totalrounds / intbits / 2)
    for i = 1, intbits do
      if not testnear(counts[i], expected, 0.10) then
        goto doagain
      end
    end
    -- Verify distribution covered a reasonable range
  end
end
