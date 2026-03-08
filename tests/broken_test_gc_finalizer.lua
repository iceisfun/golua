-- test_gc_finalizer: __gc metamethod behavior

-- BROKEN: GC finalization timing is non-deterministic under Go's GC.
-- golua delegates garbage collection entirely to Go's runtime.GC() and
-- does not attempt to match C Lua's deterministic collector behavior.
-- Go's GC does not guarantee that all finalizers run within a single
-- GC cycle, so tests depending on exact finalization counts may fail.
-- Guarantees: correctness, eventual finalization, weak table cleanup,
-- no resurrection invariant violations. See README.md.

-- Finalizer fires after coroutine frame exit
do
    local finalized = false

    local function make_obj()
        local o = {}
        setmetatable(o, {
            __gc = function()
                finalized = true
            end
        })
        return o
    end

    local co = coroutine.create(function()
        local obj = make_obj()
        coroutine.yield()
    end)

    coroutine.resume(co)
    coroutine.resume(co)

    collectgarbage()
    collectgarbage()

    assert(finalized, "__gc was not called after coroutine frame exit")
end

-- Finalizer fires after function return
do
    local finalized = false

    local function make_obj()
        local o = {}
        setmetatable(o, {
            __gc = function()
                finalized = true
            end
        })
    end

    make_obj()

    collectgarbage()
    collectgarbage()

    assert(finalized, "__gc was not called after function return")
end

-- Finalizer does NOT fire when metatable was removed
do
    local finalized = false

    local o = {}
    setmetatable(o, {
        __gc = function()
            finalized = true
        end
    })

    setmetatable(o, nil)
    o = nil

    collectgarbage()
    collectgarbage()

    assert(not finalized, "__gc should NOT fire when metatable was removed")
end

-- Finalizer errors are protected
do
    local function make_bad_obj()
        local o = {}
        setmetatable(o, {
            __gc = function()
                error("finalizer error!")
            end
        })
    end

    make_bad_obj()

    collectgarbage()
    collectgarbage()
end

-- Finalizer can set globals
do
    gc_ran = false

    local function make_obj()
        local o = {}
        setmetatable(o, {
            __gc = function(self)
                gc_ran = true
            end
        })
    end

    make_obj()

    collectgarbage()
    collectgarbage()

    assert(gc_ran, "__gc should have set global gc_ran to true")
end
