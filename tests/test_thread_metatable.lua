-- Test thread (coroutine) type-level metatable interactions

-- Bug 1: Thread __index should be dispatched via type-level metatable
do
    local co = coroutine.create(function() end)
    debug.setmetatable(co, {__index = function(t, k) return k end})
    assert(co.hello == "hello", "thread __index not dispatched: got " .. tostring(co.hello))
    assert(co[42] == 42, "thread __index not dispatched for integer key")
    -- Clean up
    debug.setmetatable(co, nil)
    print("PASS: thread __index")
end

-- Bug 2: Thread __newindex should be dispatched via type-level metatable
do
    local co = coroutine.create(function() end)
    local stored = {}
    debug.setmetatable(co, {
        __newindex = function(t, k, v) stored[k] = v end,
        __index = function(t, k) return stored[k] end
    })
    co.foo = "bar"
    assert(stored.foo == "bar", "thread __newindex not dispatched: stored.foo = " .. tostring(stored.foo))
    co[1] = "one"
    assert(stored[1] == "one", "thread __newindex not dispatched for integer key")
    -- Clean up
    debug.setmetatable(co, nil)
    print("PASS: thread __newindex")
end

-- Bug 3: Thread __eq should NOT be called (only for tables and full userdata)
do
    local co1 = coroutine.create(function() end)
    local co2 = coroutine.create(function() end)
    local eq_called = false
    debug.setmetatable(co1, {__eq = function() eq_called = true; return true end})
    -- Different threads should not be equal, and __eq should not be called
    assert(not (co1 == co2), "different threads should not be equal")
    assert(not eq_called, "thread __eq should not be called for threads")
    -- Same thread should be equal by identity without calling __eq
    eq_called = false
    assert(co1 == co1, "same thread should be equal by identity")
    assert(not eq_called, "thread __eq should not be called for identity comparison")
    -- Clean up
    debug.setmetatable(co1, nil)
    print("PASS: thread __eq not called")
end

-- Bug 4: Thread __close should work via type-level metatable for TBC
do
    local co = coroutine.create(function() end)
    local closed = false
    debug.setmetatable(co, {__close = function() closed = true end})
    do
        local x <close> = co
    end
    assert(closed, "thread __close not dispatched via type-level metatable")
    -- Clean up
    debug.setmetatable(co, nil)
    print("PASS: thread __close")
end

print("OK")
