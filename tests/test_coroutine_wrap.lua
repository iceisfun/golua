-- test_coroutine_wrap.lua
-- coroutine.wrap should return a callable that resumes the coroutine and
-- propagates yields, returns, and errors like stock Lua.

local wrapped = coroutine.wrap(function(a)
    local first = coroutine.yield(a + 1)
    return first * 2
end)

-- First call should yield a + 1
local first = wrapped(10)
assert(first == 11, string.format("expected first yield to be 11, got %s", tostring(first)))

-- Second call should resume with argument and return doubled result
local final = wrapped(20)
assert(final == 40, string.format("expected final result 40, got %s", tostring(final)))

-- Calling again after completion should error just like coroutine.wrap in Lua
local ok, err = pcall(wrapped)
assert(not ok, "expected wrapped() to error after coroutine completion")
assert(type(err) == "string" and err ~= "", "expected error string from wrapped coroutine")

-- Errors inside the coroutine should propagate to the caller of the wrapper
local boom = coroutine.wrap(function()
    error("wrapped failure")
end)

local ok2, err2 = pcall(boom)
assert(not ok2, "expected wrapped function to propagate errors")
assert(string.find(err2, "wrapped failure", 1, true), "expected error message to include coroutine failure")
