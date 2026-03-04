-- Test: calls.lua - Tail calls with varargs
-- From: calls.lua
-- What: Tests tail calls that involve vararg functions

do
  local function foo (x, ...) local a = {...}; return x, a[1], a[2] end
  local function foo1 (x) return foo(10, x, x + 1) end
  local a, b, c = foo1(-2)
  assert(a == 10 and b == -2 and c == -1)
end
