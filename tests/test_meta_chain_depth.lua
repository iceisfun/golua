-- Test that metamethod chain depth limit matches Lua 5.4 (MAXTAGLOOP=2000)
-- Lua 5.4 allows exactly 2000 __index table-to-table redirects.
-- A chain of 2001 tables (2000 hops) works, and 2002 tables (2001 hops) fails.

-- Build a chain of 2001 tables with __index pointing to the next (2000 hops)
local first = {}
local prev = first
for i = 2, 2001 do
  local t = {}
  setmetatable(prev, {__index = t})
  prev = t
end
prev.val = 42

-- Should succeed at exactly 2000 hops (2001 tables)
assert(first.val == 42, "chain of 2001 tables (2000 hops) should work")

-- Now extend to 2002 tables (2001 hops) - should fail
prev.val = nil
local extra = {}
setmetatable(prev, {__index = extra})
extra.val = 99
local ok, err = pcall(function() return first.val end)
assert(not ok, "chain of 2002 tables (2001 hops) should fail")
assert(tostring(err):find("loop"), "should mention loop in error: " .. tostring(err))

print("OK")
