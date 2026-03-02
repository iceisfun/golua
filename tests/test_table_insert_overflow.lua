-- Test: nextvar.lua - table.insert overflow wrap-around
-- From: nextvar.lua
-- What: Tests that table.insert wraps around from maxinteger to mininteger when the table's __len returns maxinteger.

do   -- testing overflow in table.insert (must wrap-around)

  local t = setmetatable({},
            {__len = function () return math.maxinteger end})
  table.insert(t, 20)
  local k, v = next(t)
  assert(k == math.mininteger and v == 20)
end
