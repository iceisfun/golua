-- Bug: table.move with float args says "number expected, got number"
-- instead of "number has no integer representation"
-- Same issue affects table.insert and table.remove with float positions.

-- table.move
local ok1, err1 = pcall(table.move, {}, 1.5, 3, 1)
assert(ok1 == false)
assert(err1:find("no integer representation"),
  "table.move float arg should say 'no integer representation', got: " .. tostring(err1))

-- table.insert
local ok2, err2 = pcall(table.insert, {1,2,3}, 1.5, 99)
assert(ok2 == false)
assert(err2:find("no integer representation"),
  "table.insert float pos should say 'no integer representation', got: " .. tostring(err2))

-- table.remove
local ok3, err3 = pcall(table.remove, {1,2,3}, 1.5)
assert(ok3 == false)
assert(err3:find("no integer representation"),
  "table.remove float pos should say 'no integer representation', got: " .. tostring(err3))

print("PASSED")
