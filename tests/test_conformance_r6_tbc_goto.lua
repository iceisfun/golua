-- BUG: __close not called when block exits via goto
-- In Lua 5.4, to-be-closed variables have __close called when the
-- block is exited by any means, including goto. GoLua skips __close
-- when goto exits the block.

-- Basic: goto out of block should trigger __close
local log = {}
do
    local x <close> = setmetatable({}, {__close = function()
        table.insert(log, "goto_close")
    end})
    goto skip
end
::skip::
assert(log[1] == "goto_close", "__close should fire on goto, got: " .. tostring(log[1]))

-- Multiple TBC vars with goto
log = {}
do
    local a <close> = setmetatable({}, {__close = function()
        table.insert(log, "a")
    end})
    local b <close> = setmetatable({}, {__close = function()
        table.insert(log, "b")
    end})
    goto done
end
::done::
assert(log[1] == "b", "b should close first (reverse), got: " .. tostring(log[1]))
assert(log[2] == "a", "a should close second, got: " .. tostring(log[2]))

-- Goto within same block doesn't close outer TBC
log = {}
do
    local x <close> = setmetatable({}, {__close = function()
        table.insert(log, "outer")
    end})
    goto inner
    ::inner::
end
assert(log[1] == "outer", "outer TBC should close at block end")
assert(#log == 1, "should only have 1 close call")
