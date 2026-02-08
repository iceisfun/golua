-- test_coroutine_running.lua
-- coroutine.running should expose the currently running thread (main or coroutine)
-- and flag whether execution is on the main thread.

local mainThread, isMain = coroutine.running()
assert(isMain == true, "expected coroutine.running() main flag true on main thread")
assert(mainThread ~= nil, "expected coroutine.running() to return main thread object")

local seenThread
local co = coroutine.create(function()
    local threadObj, flag = coroutine.running()
    assert(flag == false, "coroutine.running() inside coroutine must flag false")
    assert(threadObj ~= nil, "coroutine.running() should return coroutine object")
    seenThread = threadObj
    coroutine.yield("pause")
    local again = coroutine.running()
    assert(again == seenThread, "running() should stay stable within same coroutine")
    return "done"
end)

local ok, value = coroutine.resume(co)
assert(ok and value == "pause", "expected coroutine to yield 'pause'")

assert(seenThread ~= nil, "expected coroutine.running() to set seenThread")

local ok2, value2 = coroutine.resume(co)
assert(ok2 and value2 == "done", "expected coroutine to finish with 'done'")

-- Once dead, coroutine.running() should return the coroutine and boolean false when invoked
-- inside the coroutine. From main we cannot get the dead coroutine without a handle, so ensure
-- `coroutine.status` reports "dead".
assert(coroutine.status(co) == "dead", "expected coroutine to be dead after completion")

-- Main thread identity should remain consistent across calls.
local mainThread2, isMain2 = coroutine.running()
assert(isMain2 == true, "main flag changed unexpectedly")
assert(mainThread2 == mainThread, "main thread identity should remain stable")
