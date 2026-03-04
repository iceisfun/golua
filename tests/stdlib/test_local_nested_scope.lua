-- Test: Local variable scoping in nested blocks
-- From: locals.lua
-- What: Tests that local variables in nested blocks shadow outer variables correctly and that outer variables are restored when leaving inner blocks.

do
  local i = 10
  do local i = 100; assert(i==100) end
  do local i = 1000; assert(i==1000) end
  assert(i == 10)
  if i ~= 10 then
    local i = 20
  else
    local i = 30
    assert(i == 30)
  end
end
