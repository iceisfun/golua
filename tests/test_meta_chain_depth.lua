-- Test that metamethod chain depth limit matches Lua 5.4 (MAXTAGLOOP=2000)
-- Lua 5.4 uses `for (loop = 0; loop < MAXTAGLOOP; loop++)`, so the loop
-- runs exactly 2000 iterations. A chain of 2000 tables (1999 __index hops)
-- works, and a chain of 2001 tables (2000 hops) fails.

-- Build a chain of 2000 tables with __index pointing to the next
local first = {}
local prev = first
for i = 2, 2000 do
  local t = {}
  setmetatable(prev, {__index = t})
  prev = t
end
prev.val = 42

-- Should succeed at exactly 2000 tables deep
assert(first.val == 42, "chain of 2000 tables should work")

-- Now extend to 2001 tables - should fail
-- Remove the value from prev so it must traverse to extra
prev.val = nil
local extra = {}
setmetatable(prev, {__index = extra})
extra.val = 99
local ok, err = pcall(function() return first.val end)
assert(not ok, "chain of 2001 tables should fail")
assert(tostring(err):find("loop"), "should mention loop in error: " .. tostring(err))

print("OK")
