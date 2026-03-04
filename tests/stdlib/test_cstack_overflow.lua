-- Test: calls.lua - C-stack overflow handling
-- From: calls.lua
-- What: Tests that C-stack overflow in pcall within pcall is handled gracefully

do
  local function loop ()
    assert(pcall(loop))
  end
  local err, msg = xpcall(loop, loop)
  assert(not err and string.find(msg, "error"))
end
