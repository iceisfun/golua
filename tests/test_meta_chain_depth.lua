-- Test that metamethod chain depth limit matches Lua 5.4 (MAXTAGLOOP=2000)
-- Lua 5.4 allows a chain of 2001 tables (2000 __index hops) and
-- errors at 2002 tables (2001 hops).

-- Build a chain of 2001 tables with __index pointing to the next
local first = {}
local prev = first
for i = 2, 2001 do
  local t = {}
  setmetatable(prev, {__index = t})
  prev = t
end
prev.val = 42

-- Should succeed at exactly 2001 tables deep
assert(first.val == 42, "chain of 2001 tables should work")

-- Now extend to 2002 tables - should fail
-- Remove the value from prev so it must traverse to extra
prev.val = nil
local extra = {}
setmetatable(prev, {__index = extra})
extra.val = 99
local ok, err = pcall(function() return first.val end)
assert(not ok, "chain of 2002 tables should fail")
assert(tostring(err):find("loop"), "should mention loop in error: " .. tostring(err))

print("OK")
