-- Test: events.lua - Invalidating metamethod flags
-- From: events.lua
-- What: Tests that metamethod cache is invalidated when metamethods change

do
  local mt = {__eq = true}
  local a = setmetatable({10}, mt)
  local b = setmetatable({10}, mt)
  mt.__eq = nil
  assert(a ~= b)   -- no metamethod
  mt.__eq = function (x,y) return x[1] == y[1] end
  assert(a == b)   -- must use metamethod now
end
