-- Test: calls.lua - Tail calls with __call metamethod
-- From: calls.lua
-- What: Tests tail calls through __call metamethod chains

do
  local n = 10000
  local function foo ()
    if n == 0 then return 1023
    else n = n - 1; return foo()
    end
  end
  for i = 1, 15 do
    foo = setmetatable({}, {__call = foo})
  end
  assert(coroutine.wrap(function() return foo() end)() == 1023)
end
