-- Test: coroutine.lua - To-be-closed variables in coroutines
-- From: coroutine.lua
-- What: Tests __close metamethod behavior when closing coroutines, including error cases

do
  local function func2close (f)
    return setmetatable({}, {__close = f})
  end

  -- ok to close a dead coroutine
  local co = coroutine.create(print)
  assert(coroutine.resume(co, "testing 'coroutine.close'"))
  assert(coroutine.status(co) == "dead")
  local st, msg = coroutine.close(co)
  assert(st and msg == nil)

  -- cannot close the running coroutine
  local st, msg = pcall(coroutine.close, coroutine.running())
  assert(not st and string.find(msg, "running"))

  -- closing a coroutine after an error
  local co = coroutine.create(error)
  local st, msg = coroutine.resume(co, 100)
  assert(not st and msg == 100)
  st, msg = coroutine.close(co)
  assert(not st and msg == 100)
  st, msg = coroutine.close(co)
  assert(st and msg == nil)
end
