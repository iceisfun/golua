-- Test: nextvar.lua - Integer overflow with length operator
-- From: nextvar.lua
-- What: Tests that the length operator works on a table with keys at powers of two up to 2^50.

do
  local a = {}
  for i=0,50 do a[2^i] = true end
  assert(a[#a])
end
