-- Test: coroutine.lua - Infinite coroutine recursion
-- From: coroutine.lua
-- What: Tests that infinite recursion of coroutines is caught

do
  a = function(a) coroutine.wrap(a)(a) end
  assert(not pcall(a, a))
end
