-- Test: closure.lua - Closures x repeat-until
-- From: closure.lua
-- What: Tests that closures in repeat-until loops properly capture the loop variable

do
  local a = {}
  local i = 1
  repeat
    local x = i
    a[i] = function () i = x+1; return x end
  until i > 10 or a[i]() ~= x
  assert(i == 11 and a[1]() == 1 and a[3]() == 3 and i == 4)
end
