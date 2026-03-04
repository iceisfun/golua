-- Test: math.lua - math.random large intervals
-- From: math.lua
-- What: Tests that math.random produces diverse values over large integer ranges and covers the range adequately.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger
  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1
  local random = math.random

  do
    local function aux(p1, p2)       -- test random for large intervals
      local max = minint
      local min = maxint
      local n = 100
      local mark = {}; local count = 0   -- to count how many different values
      ::doagain::
      for _ = 1, n do
        local t = random(p1, p2)
        if not mark[t] then  -- new value
          assert(p1 <= t and t <= p2)
          max = math.max(max, t)
          min = math.min(min, t)
          mark[t] = true
          count = count + 1
        end
      end
      -- at least 80% of values are different
      if not (count >= n * 0.8) then
        goto doagain
      end
      -- min and max not too far from formal min and max
      local diff = (p2 - p1) >> 4
      if not (min < p1 + diff and max > p2 - diff) then
        goto doagain
      end
    end
    aux(0, maxint)
    aux(1, maxint)
    aux(3, maxint // 3)
    aux(minint, -1)
    aux(minint // 2, maxint // 2)
    aux(minint, maxint)
    aux(minint + 1, maxint)
    aux(minint, maxint - 1)
    aux(0, 1 << (intbits - 5))
  end
end
