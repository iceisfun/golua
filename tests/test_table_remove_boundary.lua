-- BUG: table.remove rejects pos = #t + 1
-- In Lua 5.4, table.remove accepts pos = #t + 1, returning nil
-- and erasing that position. GoLua incorrectly errors with
-- "position out of range".

-- table.remove({1,2,3}, 4) — pos = #t + 1, should return nil
local t1 = {1, 2, 3}
local r1 = table.remove(t1, 4)
assert(r1 == nil, "table.remove(t, #t+1) should return nil, got: " .. tostring(r1))
assert(#t1 == 3, "table.remove(t, #t+1) should not change length")

-- table.remove({}, 1) — empty table, pos = #t + 1 = 1
local t2 = {}
local r2 = table.remove(t2, 1)
assert(r2 == nil, "table.remove({}, 1) should return nil, got: " .. tostring(r2))
assert(#t2 == 0, "table.remove({}, 1) should not change length")

-- table.remove({}, 0) — pos = 0 when #t = 0, also valid per Lua 5.4
local t3 = {}
local r3 = table.remove(t3, 0)
assert(r3 == nil, "table.remove({}, 0) should return nil, got: " .. tostring(r3))

-- Positions that are genuinely out of range should still error
local ok4, e4 = pcall(table.remove, {1, 2, 3}, 5)
assert(not ok4, "table.remove(t, #t+2) should error")

local ok5, e5 = pcall(table.remove, {1, 2, 3}, 0)
assert(not ok5, "table.remove(t, 0) should error when #t > 0")

local ok6, e6 = pcall(table.remove, {1, 2, 3}, -1)
assert(not ok6, "table.remove(t, -1) should error")
