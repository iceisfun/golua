-- Dead errored coroutines should keep their saved level-0 C-frame locals
-- visible through debug.getlocal, matching Lua 5.4.

local co = coroutine.create(function()
    error("fail")
end)

coroutine.resume(co)
local n, v = debug.getlocal(co, 0, 1)
print(n, v)
--> =(C temporary)	fail
