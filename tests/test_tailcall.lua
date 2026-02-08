-- Test tail call optimization (should not stack overflow)
local function tail(n)
    if n == 0 then return 0 end
    return tail(n - 1)
end

assert(tail(10000) == 0)
