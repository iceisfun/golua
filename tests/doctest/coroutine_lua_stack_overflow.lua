-- Pure-Lua recursion that overflows INSIDE a single coroutine is a Lua-stack
-- overflow ("stack overflow"), not a "C stack overflow". Only resume-within-
-- resume nesting (native re-entrancy) yields "C stack overflow"
-- (see coroutine_recursive_cstack.lua).

local co = coroutine.create(function()
  local function r() return 1 + r() end
  r()
end)
local ok, err = coroutine.resume(co)
print(ok)
--> =false
print(type(err))
--> =string
-- Must be a plain Lua "stack overflow", with no leading "C ".
print(err:find("stack overflow") ~= nil)
--> =true
print(err:find("C stack overflow") ~= nil)
--> =false

-- Same via coroutine.wrap wrapped in pcall.
local w = coroutine.wrap(function()
  local function r() return 1 + r() end
  r()
end)
local ok2, err2 = pcall(w)
print(ok2)
--> =false
print(err2:find("C stack overflow") ~= nil)
--> =false
