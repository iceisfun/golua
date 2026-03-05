-- coroutine.wrap self-resume should error, not deadlock
local A
A = coroutine.wrap(function() return pcall(A, 1) end)
local st, res = A()
assert(not st)
assert(string.find(res, "non%-suspended") or string.find(res, "cannot resume"))

-- coroutine.resume on the running main thread from a child coroutine
local mainThread = coroutine.running()
local B = coroutine.create(function() return coroutine.resume(mainThread) end)
local st2, res2, res3 = coroutine.resume(B)
assert(st2 == true)   -- resume(B) itself succeeds
assert(res2 == false)  -- but resume(mainThread) inside B fails
assert(string.find(res3, "non%-suspended") or string.find(res3, "cannot resume"))
