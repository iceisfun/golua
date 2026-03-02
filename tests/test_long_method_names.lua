-- Test: calls.lua - Long method names
-- From: calls.lua
-- What: Tests that very long method names work correctly

do
  local t = {x = 1}
  function t:_012345678901234567890123456789012345678901234567890123456789 ()
    return self.x
  end
  assert(t:_012345678901234567890123456789012345678901234567890123456789() == 1)
end
