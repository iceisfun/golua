-- Test: db.lua - Coroutine debugging
-- From: db.lua
-- What: Tests debug operations (traceback, getinfo, getlocal, setlocal, sethook) on suspended coroutines

do
  local co = coroutine.create(function (x)
    local a, b = coroutine.yield(x)
    assert(a == 100 and b == nil)
    return x
  end)
  local a, b = coroutine.resume(co, 10)
  assert(a and b == 10)
  a, b = debug.getlocal(co, 1, 1)
  assert(a == "x" and b == 10)
  assert(debug.setlocal(co, 1, 1, 30) == "x")
  a, b = coroutine.resume(co, 100)
  assert(a and b == 30)
end
