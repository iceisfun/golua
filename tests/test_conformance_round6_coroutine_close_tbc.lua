-- BUG: coroutine.close() doesn't trigger __close on TBC variables
-- In Lua 5.4, when a suspended coroutine is closed via coroutine.close(),
-- all to-be-closed variables in the coroutine should have their __close
-- metamethods called. GoLua skips this.

-- Basic: coroutine.close triggers TBC __close
local log = {}
local co1 = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function()
        table.insert(log, "co_close")
    end})
    coroutine.yield()
    return "never"
end)
coroutine.resume(co1)
assert(#log == 0, "TBC should not be closed yet")
coroutine.close(co1)
assert(log[1] == "co_close", "coroutine.close should trigger __close, got: " .. tostring(log[1]))

-- Multiple TBC vars in coroutine close in reverse order
log = {}
local co2 = coroutine.create(function()
    local a <close> = setmetatable({}, {__close = function()
        table.insert(log, "a")
    end})
    local b <close> = setmetatable({}, {__close = function()
        table.insert(log, "b")
    end})
    coroutine.yield()
end)
coroutine.resume(co2)
coroutine.close(co2)
assert(log[1] == "b", "b should close first (reverse), got: " .. tostring(log[1]))
assert(log[2] == "a", "a should close second, got: " .. tostring(log[2]))

-- TBC close receives nil as error (normal close, not error)
local close_err_val = "not_set"
local co3 = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function(self, err)
        close_err_val = err
    end})
    coroutine.yield()
end)
coroutine.resume(co3)
coroutine.close(co3)
assert(close_err_val == nil, "__close should receive nil for normal close, got: " .. tostring(close_err_val))
