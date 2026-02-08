-- Test upvalue capture in for loops (each iteration gets own copy)
local functions = {}
for i = 1, 3 do
    local x = i * 10
    functions[i] = function() return x end
end

assert(functions[1]() == 10, "Closure 1 failed")
assert(functions[2]() == 20, "Closure 2 failed")
assert(functions[3]() == 30, "Closure 3 failed")

-- Test nested upvalues (3 levels deep)
local function outer()
    local val = "level1"
    return function()
        local val2 = "level2"
        return function()
            return val .. "-" .. val2
        end
    end
end

local inner = outer()()
assert(inner() == "level1-level2")
