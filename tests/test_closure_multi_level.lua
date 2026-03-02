-- Test: closure.lua - Multi-level closures
-- From: closure.lua
-- What: Tests closures that capture variables from multiple nesting levels

do
  local y, w
  function f(x)
    return function (y)
      return function (z) return w+x+y+z end
    end
  end

  y = f(10)
  w = 1.345
  assert(y(20)(30) == 60+w)
end
