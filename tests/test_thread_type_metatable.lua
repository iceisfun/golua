-- Test that debug.setmetatable on a thread sets the TYPE-LEVEL metatable
-- shared by ALL threads (existing and new), not just the one thread.

local mt = { __index = function(self, k) return "shared:" .. k end }

local co1 = coroutine.create(function() end)
local co2 = coroutine.create(function() end)

-- Set metatable on co1 — should affect ALL threads
debug.setmetatable(co1, mt)

-- Verify co1 has the metatable
local m1 = debug.getmetatable(co1)
assert(m1 == mt, "co1 should have the metatable")

-- Verify co2 also has the same metatable (type-level sharing)
local m2 = debug.getmetatable(co2)
assert(m2 == mt, "co2 should share the same type-level metatable, got: " .. tostring(m2))

-- Verify newly created threads also get the metatable
local co3 = coroutine.create(function() end)
local m3 = debug.getmetatable(co3)
assert(m3 == mt, "newly created co3 should also have the type-level metatable")

-- Verify the main thread also has the metatable
-- (The running coroutine IS a thread)

-- Setting to nil should clear for all
debug.setmetatable(co1, nil)
local m4 = debug.getmetatable(co2)
assert(m4 == nil, "after clearing, co2 should have nil metatable")

print("PASS")
