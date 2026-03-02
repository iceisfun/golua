-- Test: events.lua - Loop detection in delegation
-- From: events.lua
-- What: Tests that infinite loops in __index and __newindex chains are detected

do
  local a = {}
  setmetatable(a, a)
  a.__index = a
  a.__newindex = a
  assert(not pcall(function (a,b) return a[b] end, a, 10))
  assert(not pcall(function (a,b,c) a[b] = c end, a, 10, true))
end
