-- BUG: coroutine.close() deadlocks on suspended coroutine
-- In Lua 5.4, coroutine.close() on a suspended coroutine marks it
-- dead and returns true, nil. GoLua deadlocks (all goroutines asleep).

local co = coroutine.create(function()
    coroutine.yield(1)
    coroutine.yield(2)
    return 3
end)

-- Start the coroutine (it yields)
local ok, v = coroutine.resume(co)
assert(ok and v == 1, "first resume should yield 1")
assert(coroutine.status(co) == "suspended", "should be suspended")

-- Close the suspended coroutine
local close_ok, close_err = coroutine.close(co)
assert(close_ok == true, "coroutine.close should return true, got: " .. tostring(close_ok))
assert(close_err == nil, "coroutine.close should return nil error, got: " .. tostring(close_err))
assert(coroutine.status(co) == "dead", "closed coroutine should be dead")

-- Resuming a closed coroutine should fail
local ok2, err2 = coroutine.resume(co)
assert(not ok2, "resume after close should fail")
