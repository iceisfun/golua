-- Test: db.lua - debug.getlocal / debug.setlocal
-- From: db.lua
-- What: Tests getting and setting local variables via the debug library, including varargs

do
  assert(not pcall(debug.getlocal, 20, 1))
  assert(not pcall(debug.setlocal, -1, 1, 10))

  local function foo (a,b,...) local d, e end
  assert(debug.getlocal(foo, 1) == 'a')
  assert(debug.getlocal(foo, 2) == 'b')
  assert(not debug.getlocal(foo, 3))
end
