-- Test: upvalue integrity across coroutine yield/resume cycles

-- Basic getter/setter closures yielded from coroutine
local co = coroutine.create(function()
    local x = 10
    local function getter() return x end
    local function setter(v) x = v end
    coroutine.yield(getter, setter)
    -- After resume, x should be whatever setter set
    return x
end)

local ok, getter, setter = coroutine.resume(co)
assert(ok, "first resume should succeed")
assert(getter() == 10, "initial value should be 10, got " .. tostring(getter()))

-- Modify via setter while coroutine is suspended
setter(42)
assert(getter() == 42, "getter should return 42 after setter, got " .. tostring(getter()))

-- Resume and verify the coroutine sees the updated value
local ok2, final = coroutine.resume(co)
assert(ok2, "second resume should succeed")
assert(final == 42, "coroutine should see updated value 42, got " .. tostring(final))

-- Test: multiple yield/resume cycles with upvalue mutation
local co2 = coroutine.create(function()
    local counter = 0
    for i = 1, 5 do
        counter = counter + 1
        coroutine.yield(counter)
    end
    return counter
end)

for i = 1, 5 do
    local ok3, val = coroutine.resume(co2)
    assert(ok3, "resume " .. i .. " should succeed")
    assert(val == i, "yield " .. i .. " should produce " .. i .. ", got " .. tostring(val))
end

local ok4, final2 = coroutine.resume(co2)
assert(ok4, "final resume should succeed")
assert(final2 == 5, "final value should be 5, got " .. tostring(final2))

-- Test: shared upvalue between two closures across yield
local co3 = coroutine.create(function()
    local shared = "hello"
    local read = function() return shared end
    local write = function(v) shared = v end
    coroutine.yield(read, write)
    coroutine.yield(read, write)
    return shared
end)

local ok5, r, w = coroutine.resume(co3)
assert(ok5)
assert(r() == "hello")

w("world")
assert(r() == "world", "should read 'world' after write")

local ok6, r2, w2 = coroutine.resume(co3)
assert(ok6)
assert(r2() == "world", "second yield should still see 'world'")

w2("final")
local ok7, result = coroutine.resume(co3)
assert(ok7)
assert(result == "final", "expected 'final', got " .. tostring(result))
