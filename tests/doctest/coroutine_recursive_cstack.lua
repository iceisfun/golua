-- Recursive coroutine.create + coroutine.resume should say "C stack overflow"
-- because the overflow chain passes through native coroutine.resume frames.

local function f()
  local c = coroutine.create(f)
  local a, b = coroutine.resume(c)
  return b
end
local result = f()
print(type(result))
--> =string
print(result:find("C stack overflow") ~= nil)
--> =true
