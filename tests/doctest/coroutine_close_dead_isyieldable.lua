-- Dead coroutines stay yieldable in Lua 5.4 even after coroutine.close().
-- GoLua currently flips coroutine.isyieldable(co) to false after close.

local co = coroutine.create(function() end)
assert(coroutine.resume(co))

-- Baseline: a dead coroutine object is still yieldable as an argument.
print(coroutine.status(co), coroutine.isyieldable(co))
--> =dead	true

-- Closing the dead coroutine should preserve that result.
assert(coroutine.close(co))
print(coroutine.status(co), coroutine.isyieldable(co))
--> =dead	true
