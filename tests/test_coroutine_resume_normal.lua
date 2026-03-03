-- Test: coroutine.lua - Attempt to resume normal coroutine
-- From: coroutine.lua
-- What: Tests that a 'normal' (non-running, non-suspended) coroutine cannot be resumed

do
  local co1, co2
  co1 = coroutine.create(function () return co2() end)
  co2 = coroutine.wrap(function ()
    assert(coroutine.status(co1) == 'normal')
    assert(not coroutine.resume(co1))
    coroutine.yield(3)
  end)
  a,b = coroutine.resume(co1)
  assert(a and b == 3)
end
