-- Test coroutine argument passing
local co = coroutine.create(function(x)
    local y = coroutine.yield(x + 1)
    return y * 2
end)

local ok, a = coroutine.resume(co, 10)
assert(a == 11)

ok, b = coroutine.resume(co, 5)
assert(b == 10)
