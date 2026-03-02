-- Test: closure.lua - Closures with numeric for control variable
-- From: closure.lua
-- What: Tests that closures properly capture the for loop control variable

do
  a = {}
  for i=1,10 do
    a[i] = {set = function(x) i=x end, get = function () return i end}
    if i == 3 then break end
  end
  assert(a[4] == undef)
  a[1].set(10)
  assert(a[2].get() == 2)
  a[2].set('a')
  assert(a[3].get() == 3)
  assert(a[2].get() == 'a')
end
