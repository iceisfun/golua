-- BUG: error in __close metamethod escapes pcall
-- In Lua 5.4, errors thrown by __close metamethods are caught by
-- pcall. GoLua lets them propagate as unhandled runtime errors.

-- Basic: pcall should catch __close error
local ok, err = pcall(function()
    local x <close> = setmetatable({}, {__close = function()
        error("close_error")
    end})
end)
assert(ok == false, "pcall should catch __close error, got ok=true")
assert(type(err) == "string", "error should be a string, got: " .. type(err))
assert(err:find("close_error"), "error should contain 'close_error', got: " .. err)

-- When __close errors, other __close handlers should still run
local log = {}
local ok2, err2 = pcall(function()
    local a <close> = setmetatable({}, {__close = function()
        table.insert(log, "a")
    end})
    local b <close> = setmetatable({}, {__close = function()
        table.insert(log, "b_start")
        error("b_error")
    end})
    local c <close> = setmetatable({}, {__close = function()
        table.insert(log, "c")
    end})
end)
assert(ok2 == false, "pcall should catch __close error in multi-TBC")
assert(log[1] == "c", "c should close first (reverse order)")
assert(log[2] == "b_start", "b should close second")
assert(log[3] == "a", "a should still close even after b errors")

-- __close error replaces original error
local ok3, err3 = pcall(function()
    local x <close> = setmetatable({}, {__close = function()
        error("close_wins")
    end})
    error("original_error")
end)
assert(ok3 == false, "pcall should catch")
assert(type(err3) == "string" and err3:find("close_wins"),
    "__close error should replace original, got: " .. tostring(err3))
