-- Test _ENV override
local x = 1

local f = function()
    local _ENV = { x = 2 }
    return x
end

assert(f() == 2)
assert(x == 1)
