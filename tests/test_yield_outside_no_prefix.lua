-- Test that coroutine.yield() from outside a coroutine does not add file:line prefix
-- When called from within a Lua function wrapper, the error message should be
-- exactly "attempt to yield from outside a coroutine" with no file:line prefix.

-- Direct call (native function, no prefix expected)
local ok1, err1 = pcall(coroutine.yield)
assert(err1 == "attempt to yield from outside a coroutine",
    "direct: got: " .. tostring(err1))

-- Called from within a Lua function (should also have no prefix)
local ok2, err2 = pcall(function() coroutine.yield() end)
assert(err2 == "attempt to yield from outside a coroutine",
    "wrapped: got: " .. tostring(err2))

print("OK")
