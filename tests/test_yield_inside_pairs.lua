-- Test: nextvar.lua - yield inside __pairs
-- From: nextvar.lua
-- What: Tests that a coroutine can yield from within a __pairs metamethod and resume correctly to complete iteration.

do
  local t = setmetatable({10, 20, 30}, {__pairs = function (t)
    local inc = coroutine.yield()
    return function (t, i)
             if i > 1 then return i - inc, t[i - inc]  else return nil end
           end, t, #t + 1
  end})

  local res = {}
  local co = coroutine.wrap(function ()
    for i,p in pairs(t) do res[#res + 1] = p end
  end)

  co()     -- start coroutine
  co(1)    -- continue after yield
  assert(res[1] == 30 and res[2] == 20 and res[3] == 10 and #res == 3)

end
