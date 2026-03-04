-- Test: nextvar.lua - Length operator with negative and power-of-two indices
-- From: nextvar.lua
-- What: Tests the length operator on tables with negative indices and on tables filled to powers of two.

do
  assert(#{} == 0)
  assert(#{[-1] = 2} == 0)
  for i=0,40 do
    local a = {}
    for j=1,i do a[j]=j end
    assert(#a == i)
  end
end
