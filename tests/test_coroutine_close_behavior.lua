-- Bug 1: coroutine.close on error-dead coroutine returns true
-- instead of false + error message.
-- Bug 2: coroutine.close on suspended coroutine deadlocks (separate test).

-- Test 1: close on error-dead coroutine should return false + error
local co = coroutine.create(function() error("die") end)
local ok, err = coroutine.resume(co)
assert(not ok, "resume should fail")
assert(coroutine.status(co) == "dead", "should be dead")

local close_ok, close_err = coroutine.close(co)
assert(close_ok == false, "close on error-dead should return false, got " .. tostring(close_ok))
assert(close_err ~= nil, "close on error-dead should return error message")
assert(tostring(close_err):find("die"), "error should contain 'die', got: " .. tostring(close_err))

-- Test 2: close on successfully-dead coroutine should return true
local co2 = coroutine.create(function() return 42 end)
coroutine.resume(co2)
assert(coroutine.status(co2) == "dead")
local close_ok2, close_err2 = coroutine.close(co2)
assert(close_ok2 == true, "close on success-dead should return true, got " .. tostring(close_ok2))

-- Test 3: close on never-started coroutine should return true
local co3 = coroutine.create(function() return 1 end)
local close_ok3 = coroutine.close(co3)
assert(close_ok3 == true, "close on never-started should return true")

print("PASS")
