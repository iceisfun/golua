-- Test: gc.lua - Clearing tables
-- From: gc.lua
-- What: Tests that table entries with table keys can be iterated, removed, and
--       replaced with integer keys. Pure table manipulation test.

do
  local lim = 15
  local a = {}
  for i=1,lim do a[{}] = i end
  local b = {}
  for k,v in pairs(a) do b[k]=v end
  for n in pairs(b) do
    a[n] = nil
    assert(type(n) == 'table' and next(n) == nil)
    collectgarbage()
  end
  b = nil
  collectgarbage()
  for n in pairs(a) do error'cannot be here' end
  for i=1,lim do a[i] = i end
  for i=1,lim do assert(a[i] == i) end
end
