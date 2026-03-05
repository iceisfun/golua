-- Test: cstack.lua - Chain of coroutine.close
-- From: cstack.lua
-- What: Tests C stack overflow when closing a chain of 1000 coroutines (bug since 5.4.0)

do
  local count = 0
  local coro = false
  for i = 1, 1000 do
    local previous = coro
    coro = coroutine.create(function()
      local cc <close> = setmetatable({}, {__close=function()
        count = count + 1
        if previous then assert(coroutine.close(previous)) end
      end})
      coroutine.yield()
    end)
    assert(coroutine.resume(coro))
  end
  local st, msg = coroutine.close(coro)
  assert(not st and string.find(msg, "C stack overflow"))
end
