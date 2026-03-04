-- Test: cstack.lua - Limits in coroutines inside deep calls
-- From: cstack.lua
-- What: Tests stack limit handling when creating coroutines inside deeply recursive calls (bug in 5.4.0)

do
  local count = 0
  local lim = 1000
  local function stack (n)
    if n > 0 then return stack(n - 1) + 1
    else coroutine.wrap(function ()
      count = count + 1
      stack(lim)
    end)()
    end
  end
  local st, msg = xpcall(stack, function () return "ok" end, lim)
  assert(not st and msg == "ok")
end
