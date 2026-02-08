package main

import (
	"testing"
)

func TestGcFinalizerCoroutine(t *testing.T) {
	source := `
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
    -- obj goes out of scope here
end)

coroutine.resume(co)
coroutine.resume(co)

collectgarbage()
collectgarbage()

assert(finalized, "__gc was not called after coroutine frame exit")
`
	runLuaSource(t, source, "gc_coroutine")
}

func TestGcFinalizerFunctionReturn(t *testing.T) {
	source := `
local finalized = false

local function make_obj()
    local o = {}
    setmetatable(o, {
        __gc = function()
            finalized = true
        end
    })
    -- o goes out of scope when function returns
end

make_obj()

collectgarbage()
collectgarbage()

assert(finalized, "__gc was not called after function return")
`
	runLuaSource(t, source, "gc_function_return")
}

func TestGcFinalizerMetatableRemoved(t *testing.T) {
	source := `
local finalized = false

local o = {}
setmetatable(o, {
    __gc = function()
        finalized = true
    end
})

-- Remove metatable; when finalizer fires, it finds no __gc
setmetatable(o, nil)
o = nil

collectgarbage()
collectgarbage()

assert(not finalized, "__gc should NOT fire when metatable was removed")
`
	runLuaSource(t, source, "gc_mt_removed")
}

func TestGcFinalizerProtectedFromErrors(t *testing.T) {
	source := `
local function make_bad_obj()
    local o = {}
    setmetatable(o, {
        __gc = function()
            error("finalizer error!")
        end
    })
end

make_bad_obj()

-- Should not crash even though __gc errors
collectgarbage()
collectgarbage()
`
	runLuaSource(t, source, "gc_error_protection")
}

func TestGcFinalizerSetsGlobal(t *testing.T) {
	source := `
local function make_obj()
    local o = {}
    setmetatable(o, {
        __gc = function(self)
            gc_ran = true
        end
    })
end

gc_ran = false
make_obj()

collectgarbage()
collectgarbage()

assert(gc_ran, "__gc should have set global gc_ran to true")
`
	runLuaSource(t, source, "gc_sets_global")
}
