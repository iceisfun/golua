-- debug.setlocal on a suspended coroutine's C frame (level 0)
-- should succeed even when yield was called with zero args.
-- C Lua returns "(C temporary)" because db_setlocal passes the
-- calling thread's L (not the coroutine's) to lua_setlocal.

local co = coroutine.create(function()
  coroutine.yield()
end)
assert(coroutine.resume(co))

-- setlocal at level 0, index 1 should return "(C temporary)"
local name = debug.setlocal(co, 0, 1, 123)
assert(name == "(C temporary)", "expected '(C temporary)', got " .. tostring(name))

-- getlocal at level 0, index 1 returns nil (matching C Lua)
local gname = debug.getlocal(co, 0, 1)
assert(gname == nil, "expected nil from getlocal, got " .. tostring(gname))

print("OK")
