-- Test: coroutine.close shows "running" during __close metamethods

local co
local status_during_close
co = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function()
        status_during_close = coroutine.status(co)
    end})
    coroutine.yield()
end)
coroutine.resume(co)
coroutine.close(co)
assert(status_during_close == "running",
    "expected 'running' during __close, got '" .. tostring(status_during_close) .. "'")

-- After close, status should be dead
assert(coroutine.status(co) == "dead",
    "expected 'dead' after close, got '" .. coroutine.status(co) .. "'")

print("OK")
