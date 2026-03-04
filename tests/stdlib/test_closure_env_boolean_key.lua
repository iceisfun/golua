-- Test: closure.lua - Bug in 5.4.7: _ENV boolean key access
-- From: closure.lua
-- What: Tests that _ENV[true] works correctly through a closure

do
  _ENV[true] = 10
  local function aux () return _ENV[1 < 2] end
  assert(aux() == 10)
  _ENV[true] = nil
end
