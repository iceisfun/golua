-- Test: closure.lua - Closure with break in for loop
-- From: closure.lua
-- What: Tests that closures capture the correct value when break exits a for loop

do
  local f
  for i=1,3 do
    f = function () return i end
    break
  end
  assert(f() == 1)
end
