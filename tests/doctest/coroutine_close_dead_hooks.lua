-- Hook state on a coroutine should survive coroutine.close(), and hooks can
-- still be queried or replaced on the closed thread object in Lua 5.4.

local co = coroutine.create(function() end)
local hook = function() end
debug.sethook(co, hook, "c")
assert(coroutine.resume(co))
assert(coroutine.close(co))

-- Existing hook state should still be visible after close.
local f1, mask1, count1 = debug.gethook(co)
print(type(f1), mask1, count1)
--> =function	c	0

-- Installing a new hook after close should also work.
debug.sethook(co, hook, "cr", 7)
local f2, mask2, count2 = debug.gethook(co)
print(type(f2), mask2, count2)
--> =function	cr	7
