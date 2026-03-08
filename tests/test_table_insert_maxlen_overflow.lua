-- Test: table.insert with __len returning math.maxinteger should not overflow
-- Bug: pos > length+1 overflows when length == math.maxinteger

local t = setmetatable({}, {
  __len = function() return math.maxinteger end,
  __newindex = function() end,
  __index = function() return nil end,
})

-- Position 1 should be valid (1 <= 1 <= maxinteger+1, but +1 overflows)
local ok, err = pcall(table.insert, t, 1, "x")
assert(ok, "table.insert pos=1 should succeed, got: " .. tostring(err))

-- Position math.maxinteger should also be valid
local ok2, err2 = pcall(table.insert, t, math.maxinteger, "y")
assert(ok2, "table.insert pos=maxinteger should succeed, got: " .. tostring(err2))

-- Position 0 should still fail
local ok3 = pcall(table.insert, t, 0, "z")
assert(not ok3, "table.insert pos=0 should fail")

print("PASS")
