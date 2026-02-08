-- Test varargs with select()
local function sum(...)
    local s = 0
    for i = 1, select("#", ...) do
        s = s + select(i, ...)
    end
    return s
end

assert(sum() == 0)
assert(sum(1) == 1)
assert(sum(1,2,3,4) == 10)
