-- Test modifying upvalue from closure
local x = 10

local function f()
    x = x + 1
end

f()
assert(x == 11)
