-- Test: coroutine.lua - Coroutine errors and status
-- From: coroutine.lua
-- What: Tests error propagation in coroutines, dead coroutine status, and resume-after-error behavior

do
  function foo ()
    coroutine.yield(3)
    error(foo)
  end
  function goo() foo() end
  x = coroutine.wrap(goo)
  assert(x() == 3)
  local a,b = pcall(x)
  assert(not a and b == foo)

  x = coroutine.create(goo)
  a,b = coroutine.resume(x)
  assert(a and b == 3)
  a,b = coroutine.resume(x)
  assert(not a and b == foo and coroutine.status(x) == "dead")
end
