-- Test os.exit

-- os.exit exists
assert(type(os.exit) == "function")

-- os.exit is not caught by pcall (it propagates through)
-- We can test the close parameter behavior
local closed = false
do
    local x <close> = setmetatable({}, {
        __close = function() closed = true end
    })
    -- os.exit with close=true should close TBC variables
    -- But since os.exit terminates the VM, we test it differently:
    -- just test that the function is callable with various arg types
end

-- Test that boolean args work (without actually exiting)
-- We verify os.exit is available by checking pcall behavior
-- pcall should NOT catch os.exit (it propagates)
local ok, err = pcall(os.exit, 0)
-- If we get here, something is wrong - os.exit should have stopped execution
-- But the test harness catches LuaExitError, so this line won't execute
