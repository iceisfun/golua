-- Test: goto.lua - Bug in 5.2-5.3.2 SETNIL joining
-- From: goto.lua
-- What: Tests that a local variable declared after a label gets properly re-initialized on backward goto

do
  local x
  ::L1::
  local y
  assert(y == nil)
  y = true
  if x == nil then
    x = 1
    goto L1
  else
    x = x + 1
  end
  assert(x == 2 and y == true)
end
