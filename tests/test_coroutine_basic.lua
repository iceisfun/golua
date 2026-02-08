-- Test basic coroutine yield/resume
local co = coroutine.create(function()
    coroutine.yield(1)
    coroutine.yield(2)
    return 3
end)

local ok, v = coroutine.resume(co)
assert(ok and v == 1)

ok, v = coroutine.resume(co)
assert(ok and v == 2)

ok, v = coroutine.resume(co)
assert(ok and v == 3)

assert(coroutine.status(co) == "dead")
