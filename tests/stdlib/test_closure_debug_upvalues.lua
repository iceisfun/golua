-- Test: closure.lua - Debug manipulation of upvalues
-- From: closure.lua
-- What: Tests debug.upvalueid, debug.upvaluejoin for sharing and comparing upvalues between closures

do
  local debug = require'debug'
  local foo1, foo2, foo3
  do
    local a , b, c = 3, 5, 7
    foo1 = function () return a+b end;
    foo2 = function () return b+a end;
    do
      local a = 10
      foo3 = function () return a+b end;
    end
  end

  assert(debug.upvalueid(foo1, 1) == debug.upvalueid(foo2, 2))
  assert(debug.upvalueid(foo1, 2) == debug.upvalueid(foo2, 1))
  assert(debug.upvalueid(foo1, 1) ~= debug.upvalueid(foo3, 1))

  assert(foo1() == 3 + 5 and foo2() == 5 + 3)
  debug.upvaluejoin(foo1, 2, foo2, 2)
  assert(foo1() == 3 + 3 and foo2() == 5 + 3)
end
