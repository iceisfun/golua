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

do
  local t = coroutine.running()
  assert(select('#', debug.getlocal(t, 0, 1)) == 2)
  assert(debug.getlocal(t, 0, 1) == '(C temporary)')
  assert(debug.setlocal(t, 0, 1, 123) == '(C temporary)')
  assert(debug.setlocal(0, 1, 456) == '(C temporary)')
end

do
  local co = coroutine.create(function()
    local x = 1
    coroutine.yield()
    return x
  end)

  assert(coroutine.resume(co))
  local name, val = debug.getlocal(co, 1, 1)
  assert(name == 'x' and val == 1)
  assert(debug.setlocal(co, 1, 1, 42) == 'x')
  local ok, ret = coroutine.resume(co)
  assert(ok and ret == 42)
end

do
  local co = coroutine.create(function()
    local x = 1
    coroutine.yield()
  end)

  assert(coroutine.resume(co))
  assert(select('#', debug.getlocal(co, 1, 99)) == 1)
  assert(debug.getlocal(co, 1, 99) == nil)
  assert(select('#', debug.setlocal(co, 1, 99, 'x')) == 1)
  assert(debug.setlocal(co, 1, 99, 'x') == nil)

  assert(select('#', debug.getlocal(co, 0, 1)) == 1)
  assert(debug.getlocal(co, 0, 1) == nil)
  assert(select('#', debug.getlocal(co, 0, 99)) == 1)
  assert(debug.getlocal(co, 0, 99) == nil)
end

do
  local co = coroutine.create(function() end)
  local ok, msg = pcall(debug.getlocal, co, 1, 1)
  assert(not ok and string.find(msg, 'level out of range'))
  ok, msg = pcall(debug.setlocal, co, 1, 1, 10)
  assert(not ok and string.find(msg, 'level out of range'))
end

do
  local co = coroutine.create(function()
    local x = 1
    coroutine.yield()
  end)

  assert(coroutine.resume(co))
  assert(coroutine.resume(co))
  local ok, msg = pcall(debug.getlocal, co, 1, 1)
  assert(not ok and string.find(msg, 'level out of range'))
  ok, msg = pcall(debug.setlocal, co, 1, 1, 10)
  assert(not ok and string.find(msg, 'level out of range'))
end

do
  local co = coroutine.create(function() end)
  local function foo(a, b, ...) local c end
  assert(debug.getlocal(co, foo, 1) == 'a')
  assert(debug.getlocal(co, foo, 2) == 'b')
  assert(select('#', debug.getlocal(co, foo, 3)) == 1)
  assert(debug.getlocal(co, foo, 3) == nil)

  local ok, msg = pcall(debug.setlocal, co, foo, 1, 10)
  assert(not ok and string.find(msg, 'number expected'))
end

do
  local co = coroutine.create(function()
    local t = coroutine.running()
    local name = debug.getlocal(t, 0, 1)
    assert(name == '(C temporary)')
  end)
  assert(coroutine.resume(co))
end
