-- test_gc_setmetatable_finalizer: __gc fires after a single collectgarbage()
-- on a setmetatable'd table whose only Lua-side references were a discarded
-- temporary return value from setmetatable.
--
-- Regression for the vm.retBuf pinning bug (fuzz campaign 2026-05-10):
-- the per-VM retBuf scratch slice retained a reference to the returned
-- table across the subsequent collectgarbage() call, so Go's GC saw the
-- table as reachable and the __gc handler never fired. Reference Lua 5.5
-- prints "true" for each variant below.

-- Variant 1: bare setmetatable + collectgarbage
do
    local fired = false
    setmetatable({}, {__gc = function() fired = true end})
    collectgarbage()
    assert(fired, "v1: __gc should fire after single collectgarbage()")
end

-- Variant 2: do-block scoping
do
    local fired = false
    do
        setmetatable({}, {__gc = function() fired = true end})
    end
    collectgarbage()
    assert(fired, "v2: __gc should fire after scope exit + collectgarbage()")
end

-- Variant 3: indirection via function return
do
    local fired = {false}
    local function mk(state)
        setmetatable({}, {__gc = function() state[1] = true end})
    end
    mk(fired)
    collectgarbage()
    assert(fired[1], "v3: __gc should fire after function returns + collectgarbage()")
end

-- Variant 4: many subsequent locals don't change behavior
do
    local fired = false
    setmetatable({}, {__gc = function() fired = true end})
    local a, b, c, d, e, f, g, h = 1, 2, 3, 4, 5, 6, 7, 8
    local _ = a + b + c + d + e + f + g + h
    collectgarbage()
    assert(fired, "v4: __gc should fire even after intervening local declarations")
end

print("test_gc_setmetatable_finalizer: OK")
