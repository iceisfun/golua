-- Test: cstack.lua - Stack overflow in message handling
-- From: cstack.lua
-- What: Tests that xpcall with a handler that also overflows the stack produces "error in error handling"

do
  local count = 0
  local function loop (x, y, z)
    count = count + 1
    return 1 + loop(x, y, z)
  end
  local res, msg = xpcall(loop, loop)
  assert(msg == "error in error handling")
end
