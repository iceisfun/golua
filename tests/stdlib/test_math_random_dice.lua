-- Test: math.lua - math.random dice distribution
-- From: math.lua
-- What: Tests that math.random(6) produces a roughly uniform distribution across all six faces.

do
  local random = math.random

  local function testnear (val, ref, tol)
    return math.abs(val - ref) <= ref * tol
  end

  do
    -- test distribution for a dice
    local count = {0, 0, 0, 0, 0, 0}
    local rep = 200
    local totalrep = 0
    ::doagain::
    for i = 1, rep * 6 do
      local r = random(6)
      count[r] = count[r] + 1
    end
    totalrep = totalrep + rep
    for i = 1, 6 do
      if not testnear(count[i], totalrep, 0.05) then
        goto doagain
      end
    end
  end
end
