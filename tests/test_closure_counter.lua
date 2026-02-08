-- Test closures with independent upvalues
local function make_counter()
    local i = 0
    return function()
        i = i + 1
        return i
    end
end

local c1 = make_counter()
local c2 = make_counter()

assert(c1() == 1)
assert(c1() == 2)
assert(c2() == 1)
