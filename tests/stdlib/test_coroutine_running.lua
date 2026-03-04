-- Test: coroutine.lua - Basic coroutine.running
-- From: coroutine.lua
-- What: Tests coroutine.running returns the main thread with ismain=true

do
  local main, ismain = coroutine.running()
  assert(type(main) == "thread" and ismain)
  assert(not coroutine.resume(main))
  assert(not coroutine.isyieldable(main) and not coroutine.isyieldable())
  assert(not pcall(coroutine.yield))
end
