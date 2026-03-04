-- Test: events.lua - __call metamethod
-- From: events.lua
-- What: Tests calling a table via __call metamethod

do
  local t = {}
  local function f (t, ...) return t, {...} end
  t.__call = f

  local a = setmetatable({}, t)
  local x, y = a(table.unpack{'a', 1})
  assert(x == a and y[1] == 'a' and y[2] == 1 and y[3] == nil)
  x, y = a()
  assert(x == a and y[1] == nil)
end
