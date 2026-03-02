-- Test: table.move basics
-- From: sort.lua
-- What: Tests table.move for forward moves, backward moves, moves to new tables, overlapping moves, empty moves, and moves to the same position.

do
  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  checkerror("table expected", table.move, 1, 2, 3, 4)

  local function eqT (a, b)
    for k, v in pairs(a) do assert(b[k] == v) end
    for k, v in pairs(b) do assert(a[k] == v) end
  end

  local a = table.move({10,20,30}, 1, 3, 2)  -- move forward
  eqT(a, {10,10,20,30})

  -- move forward with overlap of 1
  a = table.move({10, 20, 30}, 1, 3, 3)
  eqT(a, {10, 20, 10, 20, 30})

  -- moving to the same table (not being explicit about it)
  a = {10, 20, 30, 40}
  table.move(a, 1, 4, 2, a)
  eqT(a, {10, 10, 20, 30, 40})

  a = table.move({10,20,30}, 2, 3, 1)   -- move backward
  eqT(a, {20,30,30})

  a = {}   -- move to new table
  assert(table.move({10,20,30}, 1, 3, 1, a) == a)
  eqT(a, {10,20,30})

  a = {}
  assert(table.move({10,20,30}, 1, 0, 3, a) == a)  -- empty move (no move)
  eqT(a, {})

  a = table.move({10,20,30}, 1, 10, 1)   -- move to the same place
  eqT(a, {10,20,30})
end
