-- Test: coroutine.lua - Multiple yield/resume arguments
-- From: coroutine.lua
-- What: Tests passing multiple arguments through yield and resume

do
  local f
  local function foo (a, ...)
    local x, y = coroutine.running()
    assert(x == f and y == false)
    assert(coroutine.resume(f) == false)
    assert(coroutine.status(f) == "running")
    local arg = {...}
    for i=1,#arg do
      _G.x = {coroutine.yield(table.unpack(arg[i]))}
    end
    return table.unpack(a)
  end

  f = coroutine.create(foo)
  assert(type(f) == "thread" and coroutine.status(f) == "suspended")
  local s,a,b,c,d
  s,a,b,c,d = coroutine.resume(f, {1,2,3}, {}, {1}, {'a', 'b', 'c'})
  assert(s and a == nil and coroutine.status(f) == "suspended")
end
