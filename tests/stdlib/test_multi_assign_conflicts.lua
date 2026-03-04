-- Test: attrib.lua - Multiple assignment conflicts
-- From: attrib.lua
-- What: Tests that multiple assignment evaluates all RHS before any LHS assignment

do
  local a,i,j,b
  a = {'a', 'b'}; i=1; j=2; b=a
  i, a[i], a, j, a[j], a[i+j] = j, i, i, b, j, i
  assert(i == 2 and b[1] == 1 and a == 1 and j == b and b[2] == 2 and
         b[3] == 1)
end
