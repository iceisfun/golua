-- Test: attrib.lua - Table constructor and indexing
-- From: attrib.lua
-- What: Tests table creation, boolean keys, large float/integer indices, constructor syntax

do
  local a = {}
  a[true] = 20
  a[false] = 10
  assert(a[1<2] == 20 and a[1>2] == 10)

  local a = {}
  for i=3000,-3000,-1 do a[i + 0.0] = i; end
  a[10e30] = "alo"; a[true] = 10; a[false] = 20
  assert(a[10e30] == 'alo' and a[not 1] == 20 and a[10<20] == 10)
  for i=3000,-3000,-1 do assert(a[i] == i); end
end
