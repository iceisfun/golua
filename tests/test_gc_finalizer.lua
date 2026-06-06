-- test_gc_finalizer: __gc metamethod behavior
--
-- Previously broken: an object whose last reference was a leftover register in
-- a completed coroutine's stack/retBuf was pinned by the coroutine VM, so its
-- __gc never fired even after collectgarbage(). Fixed by releasing a finished
-- coroutine VM's stack and retBuf (vm.ReleaseDeadStack), matching reference Lua
-- which frees a dead coroutine's stack.

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
