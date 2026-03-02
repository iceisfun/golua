-- Test: events.lua - Chained __index metatables
-- From: events.lua
-- What: Tests three levels of __index chaining

do
  local a;
  a = setmetatable({}, {__index = setmetatable({},
                   {__index = setmetatable({},
                   {__index = function (_,n) return a[n-3]+4, "lixo" end})})})
  a[0] = 20
  for i=0,10 do
    assert(a[i*3] == 20 + i*4)
  end
end
