-- setmetatable must require exactly 2 arguments
-- Bug: setmetatable(t) with missing 2nd arg silently cleared the metatable

-- Missing second argument should error
local t = setmetatable({}, {__index = function() return 42 end})
local ok, err = pcall(setmetatable, t)
assert(not ok, "setmetatable(t) with missing arg should error")
assert(string.find(err, "nil or table expected"), "wrong error: " .. tostring(err))

-- Verify the metatable was NOT cleared (the pcall caught the error)
assert(t.anything == 42, "metatable should still be intact")

-- Explicit nil second argument should still work
local t2 = setmetatable({}, {__index = function() return 99 end})
setmetatable(t2, nil)
assert(getmetatable(t2) == nil, "explicit nil should clear metatable")

-- Normal setmetatable should still work
local t3 = {}
local mt = {__index = function() return 77 end}
setmetatable(t3, mt)
assert(t3.x == 77)

print("PASS")
