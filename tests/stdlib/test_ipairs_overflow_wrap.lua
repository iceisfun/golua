-- Test: nextvar.lua - ipairs overflow wrap-around
-- From: nextvar.lua
-- What: Tests that the ipairs iterator function wraps from maxinteger to mininteger.

do   -- overflow (must wrap-around)
  local f = ipairs{}
  local k, v = f({[math.mininteger] = 10}, math.maxinteger)
  assert(k == math.mininteger and v == 10)
  k, v = f({[math.mininteger] = 10}, k)
  assert(k == nil)
end
