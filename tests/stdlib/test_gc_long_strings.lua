-- Test: gc.lua - Long strings
-- From: gc.lua
-- What: Tests creating and manipulating long strings via repeated concatenation and gsub.
--       Checks string length correctness.

do
  print('long strings')
  local x = "01234567890123456789012345678901234567890123456789012345678901234567890123456789"
  assert(string.len(x)==80)
  local s = ''
  local k = math.min(300, (math.maxinteger // 80) // 2)
  for n = 1, k do s = s..x; local j=tostring(n)  end
  assert(string.len(s) == k*80)
  s = string.sub(s, 1, 10000)
  local s, i = string.gsub(s, '(%d%d%d%d)', '')
  assert(i==10000 // 4)
end
