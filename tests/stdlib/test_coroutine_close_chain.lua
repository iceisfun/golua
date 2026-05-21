-- Test: cstack.lua - Chain of coroutine.close
-- From: cstack.lua
-- What: closing a deep chain of coroutines must produce a *catchable* stack
-- overflow, never crash the host. The chain length must exceed the VM's
-- DefaultMaxCallDepth (10000); reference Lua overflows much sooner via its
-- separate LUAI_MAXCCALLS C-call limit, but golua uses one unified depth.

do
  local count = 0
  local coro = false
  for i = 1, 25000 do
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
  assert(not st and string.find(msg, "stack overflow"))
end
