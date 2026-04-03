-- Test: closure.lua - Closures with numeric for control variable
-- From: closure.lua (updated for Lua 5.5)
-- What: Tests that closures properly capture the for loop control variable.
-- In Lua 5.5, for-loop control variables are read-only, so closures can
-- only read (get) them, not set them. We shadow with a local to test
-- mutable capture behavior.

do
  a = {}
  for i=1,10 do
    local ii = i  -- shadow with mutable local
    a[i] = {set = function(x) ii=x end, get = function () return ii end}
    if i == 3 then break end
  end
  assert(a[4] == undef)
  a[1].set(10)
  assert(a[2].get() == 2)
  a[2].set('a')
  assert(a[3].get() == 3)
  assert(a[2].get() == 'a')
end
