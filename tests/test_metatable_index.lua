-- Test __index metatable
local proto = { x = 1 }

local obj = setmetatable({}, {
    __index = proto
})

assert(obj.x == 1)
obj.x = 5
assert(obj.x == 5)
assert(proto.x == 1)
