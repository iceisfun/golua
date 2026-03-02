-- Test: table.move with extreme indices
-- From: sort.lua
-- What: Tests table.move with maxinteger/mininteger boundary indices, moves between extreme ranges, and metamethods on source/destination tables.

do
  local maxI = math.maxinteger
  local minI = math.mininteger

  local function eqT (a, b)
    for k, v in pairs(a) do assert(b[k] == v) end
    for k, v in pairs(b) do assert(a[k] == v) end
  end

  -- moving on the fringes
  local a = table.move({[maxI - 2] = 1, [maxI - 1] = 2, [maxI] = 3},
                 maxI - 2, maxI, -10, {})
  eqT(a, {[-10] = 1, [-9] = 2, [-8] = 3})

  a = table.move({[minI] = 1, [minI + 1] = 2, [minI + 2] = 3},
                 minI, minI + 2, -10, {})
  eqT(a, {[-10] = 1, [-9] = 2, [-8] = 3})

  a = table.move({45}, 1, 1, maxI)
  eqT(a, {45, [maxI] = 45})

  a = table.move({[maxI] = 100}, maxI, maxI, minI)
  eqT(a, {[minI] = 100, [maxI] = 100})

  a = table.move({[minI] = 100}, minI, minI, maxI)
  eqT(a, {[minI] = 100, [maxI] = 100})
end
