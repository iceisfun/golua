-- Test __newindex metatable
local log = {}

local t = setmetatable({}, {
    __newindex = function(_, k, v)
        log[k] = v
    end
})

t.a = 10
t.b = 20

assert(log.a == 10)
assert(log.b == 20)
