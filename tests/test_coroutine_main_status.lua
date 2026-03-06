-- coroutine.status should report correct status for the main thread
-- Bug: main thread always reported as "dead"

local main = coroutine.running()
assert(main ~= nil, "coroutine.running() should return main thread")

-- Main thread should be "running" when checked from main
assert(coroutine.status(main) == "running",
    "main thread status should be 'running', got '" .. coroutine.status(main) .. "'")

-- From inside a coroutine, main should be "normal"
local co = coroutine.create(function()
    coroutine.yield(coroutine.status(main))
end)
local ok, mainStatusFromCo = coroutine.resume(co)
assert(ok)
assert(mainStatusFromCo == "normal",
    "main thread status from coroutine should be 'normal', got '" .. tostring(mainStatusFromCo) .. "'")

-- After coroutine finishes, main should still be "running"
coroutine.resume(co)  -- finish the coroutine
assert(coroutine.status(main) == "running",
    "main thread should still be 'running' after coroutine finishes")

-- Nested: main is "normal" even from nested coroutine
local co2 = coroutine.create(function()
    local co3 = coroutine.create(function()
        coroutine.yield(coroutine.status(main))
    end)
    local ok2, status = coroutine.resume(co3)
    assert(ok2)
    coroutine.yield(status)
end)
ok, mainStatusFromNested = coroutine.resume(co2)
assert(ok)
assert(mainStatusFromNested == "normal",
    "main thread from nested coroutine should be 'normal', got '" .. tostring(mainStatusFromNested) .. "'")

print("PASS")
