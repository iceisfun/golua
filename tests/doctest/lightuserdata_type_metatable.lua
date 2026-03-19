-- Lightuserdata values from debug.upvalueid() should honor type metatables.
-- Lua 5.4 lets getmetatable/debug.setmetatable round-trip these values.

local function outer()
    local x = 1
    return function() return x end
end

local id = debug.upvalueid(outer(), 1)
debug.setmetatable(id, { tag = 123 })
local mt = getmetatable(id)

print(type(id), mt == nil, mt and mt.tag)
--> =userdata	false	123
