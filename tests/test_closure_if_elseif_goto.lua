-- Test: closure.lua - Closures in if/elseif/else with goto
-- From: closure.lua
-- What: Tests closures created in different branches of if/elseif/else with goto labels

do
  a = {}
  for i = 1, 10 do
    if i % 3 == 0 then
      local y = 0
      a[i] = function (x) local t = y; y = x; return t end
    elseif i % 3 == 1 then
      goto L1
      ::L1::
      local y = 1
      a[i] = function (x) local t = y; y = x; return t end
    elseif i % 3 == 2 then
      local t
      goto l4
      ::l4a:: a[i] = t; goto l4b
      ::l4::
      local y = 2
      t = function (x) local t = y; y = x; return t end
      goto l4a
      ::l4b::
    end
  end

  for i = 1, 10 do
    assert(a[i](i * 10) == i % 3 and a[i]() == i * 10)
  end
end
