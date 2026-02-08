-- Test __gc finalization

-- 1. Basic __gc after function return
local finalized1 = false
local function make_obj1()
    local o = {}
    setmetatable(o, {
        __gc = function()
            finalized1 = true
        end
    })
end
make_obj1()
collectgarbage()
collectgarbage()
assert(finalized1, "Test 1 failed: __gc not called after function return")

-- 2. __gc with coroutine interaction
local finalized2 = false
local co = coroutine.create(function()
    local o = {}
    setmetatable(o, {
        __gc = function()
            finalized2 = true
        end
    })
    coroutine.yield()
end)
coroutine.resume(co)
coroutine.resume(co)
collectgarbage()
collectgarbage()
assert(finalized2, "Test 2 failed: __gc not called after coroutine exit")

-- 3. __gc error handling (should not crash)
local function make_bad_obj()
    local o = {}
    setmetatable(o, {
        __gc = function()
            error("gc error!")
        end
    })
end
make_bad_obj()
collectgarbage()  -- should not crash

-- 4. __gc sets a global
gc_marker = false
local function make_obj4()
    local o = {}
    setmetatable(o, {
        __gc = function(self)
            gc_marker = true
        end
    })
end
make_obj4()
collectgarbage()
collectgarbage()
assert(gc_marker, "Test 4 failed: __gc did not set global")

-- 5. __gc not called when metatable removed before collection
local finalized5 = false
local o5 = {}
setmetatable(o5, {
    __gc = function()
        finalized5 = true
    end
})
setmetatable(o5, nil)
o5 = nil
collectgarbage()
collectgarbage()
assert(not finalized5, "Test 5 failed: __gc called after metatable removal")
