-- Test: nextvar.lua - ipairs with false values
-- From: nextvar.lua
-- What: Tests that ipairs correctly iterates through false values without stopping.

do
  local x = false
  local i = 0
  for k,v in ipairs{true,false,true,false} do
    i = i + 1
    x = not x
    assert(x == v)
  end
  assert(i == 4)
end
