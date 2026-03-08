-- Test: table.remove bounds check overflow when __len returns math.maxinteger

local t = setmetatable({}, {
  __len = function() return math.maxinteger end,
  __newindex = function() end,
  __index = function() return nil end,
})

-- Default position (== length) should succeed without bounds check overflow
local ok, err = pcall(table.remove, t)
assert(ok, "table.remove default pos should succeed, got: " .. tostring(err))

-- Position == length should succeed
local ok2, err2 = pcall(table.remove, t, math.maxinteger)
assert(ok2, "table.remove pos=maxinteger should succeed, got: " .. tostring(err2))

-- Position 0 should still fail
local ok3 = pcall(table.remove, t, 0)
assert(not ok3, "table.remove pos=0 should fail")

print("PASS")
