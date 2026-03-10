-- Round 20 debug library fixes

-- Fix 1: debug.gethook() returns 1 value (nil) when no hook set
do
    local n = select('#', debug.gethook())
    assert(n == 1, "gethook with no hook should return 1 value, got " .. n)
end

-- Fix 2: debug.getinfo invalid option error message (no '%c' suffix)
do
    local ok, err = pcall(debug.getinfo, 1, "x")
    assert(not ok)
    assert(err:match("invalid option%)$"), "expected 'invalid option)' at end, got: " .. tostring(err))
end

-- Fix 3: coroutine traceback should NOT have trailing [C]: in ?
do
    -- Simple yielded coroutine
    local co = coroutine.create(function() coroutine.yield() end)
    coroutine.resume(co)
    local tb = debug.traceback(co)
    assert(not tb:match("%[C%]: in %?$"), "simple coroutine traceback should not end with [C]: in ?, got:\n" .. tb)

    -- Coroutine with message
    local tb2 = debug.traceback(co, "msg", 0)
    assert(not tb2:match("%[C%]: in %?"), "coroutine traceback with msg should not have [C]: in ?, got:\n" .. tb2)

    -- Coroutine with nested function calls
    local co2 = coroutine.create(function()
        local function inner() coroutine.yield() end
        inner()
    end)
    coroutine.resume(co2)
    local tb3 = debug.traceback(co2, "msg", 0)
    assert(not tb3:match("%[C%]: in %?$"), "nested coroutine traceback should not end with [C]: in ?, got:\n" .. tb3)

    -- Coroutine with pcall
    local co3 = coroutine.create(function()
        pcall(function()
            coroutine.yield()
        end)
    end)
    coroutine.resume(co3)
    local tb4 = debug.traceback(co3, "msg", 0)
    assert(not tb4:match("%[C%]: in %?$"), "pcall coroutine traceback should not end with [C]: in ?, got:\n" .. tb4)

    -- Main thread traceback SHOULD still have [C]: in ?
    local tb_main = debug.traceback("msg", 0)
    assert(tb_main:match("%[C%]: in %?$"), "main thread traceback should end with [C]: in ?, got:\n" .. tb_main)
end

print("PASS: all debug r20 tests")
