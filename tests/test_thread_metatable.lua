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

-- Bug 5: Indexing a thread from library code (ipairs, table.unpack,
-- table.concat, table.insert) or through an __index/__newindex chain must use
-- the thread type metatable, not the thread's empty backing table.
do
    local co = coroutine.create(function() end)
    local store = {}
    debug.setmetatable(co, {
        __index = function(_, k) return store[k] or ("T:" .. tostring(k)) end,
        __newindex = function(_, k, v) store[k] = v end,
        __len = function() return 2 end,
    })

    local seen = {}
    for i, v in ipairs(co) do
        seen[#seen + 1] = v
        if i >= 2 then break end
    end
    assert(seen[1] == "T:1" and seen[2] == "T:2", "ipairs over thread")
    local a, b = table.unpack(co)
    assert(a == "T:1" and b == "T:2", "table.unpack over thread")
    assert(table.concat(co) == "T:1T:2", "table.concat over thread")

    -- __index chain passing through a thread
    local viaIndex = setmetatable({}, {__index = co})
    assert(viaIndex.x == "T:x", "__index = thread: got " .. tostring(viaIndex.x))
    local sm = getmetatable("")
    local oldIndex = sm.__index
    sm.__index = co
    local ok, r = pcall(function() return ("s").y end)
    sm.__index = oldIndex
    assert(ok and r == "T:y", "string __index = thread: " .. tostring(r))

    -- __newindex chain passing through a thread
    local viaNewindex = setmetatable({}, {__newindex = co})
    viaNewindex.k = "v"
    assert(store.k == "v" and rawget(viaNewindex, "k") == nil, "__newindex = thread")
    sm.__newindex = co
    local proxy = setmetatable({}, {__newindex = "str"})
    ok, r = pcall(function() proxy.w = "z" end)
    sm.__newindex = nil
    assert(ok and store.w == "z", "string __newindex = thread: " .. tostring(r))

    -- library writes through a thread __newindex
    table.move({"m1", "m2"}, 1, 2, 5, setmetatable({}, {__newindex = co}))
    assert(store[5] == "m1" and store[6] == "m2", "table.move into __newindex = thread")

    debug.setmetatable(co, nil)
    assert(not pcall(table.unpack, co), "thread without metatable is not indexable")
    print("PASS: thread metatable from library code and __index chains")
end

-- Bug 6: A method call on a thread (co:m()) must resolve through the thread
-- type metatable exactly like the field read co.m does.
do
    local co = coroutine.create(function() end)
    local ok, err = pcall(function() return co:status() end)
    assert(not ok and err:find("attempt to index a thread value %(upvalue 'co'%)"),
        "method call on a thread without metatable: " .. tostring(err))
    debug.setmetatable(co, {__index = coroutine})
    assert(co.status(co) == "suspended", "field read through thread __index")
    assert(co:status() == "suspended", "method call through thread __index")
    -- A string __index resolves string.len, which then rejects the thread.
    debug.setmetatable(co, {__index = "abc"})
    ok, err = pcall(function() return co:len() end)
    assert(not ok and err:find("bad self"), "thread __index = string: " .. tostring(err))
    -- A string method call whose __index is a thread chains through it too.
    debug.setmetatable(co, {__index = {foo = function(s) return "foo:" .. s end}})
    local sm = getmetatable("")
    local oldIndex = sm.__index
    sm.__index = co
    ok, err = pcall(function() return ("x"):foo() end)
    sm.__index = oldIndex
    assert(ok and err == "foo:x", "string method through __index = thread: " .. tostring(err))
    debug.setmetatable(co, nil)
    print("PASS: thread method call")
end

print("OK")
