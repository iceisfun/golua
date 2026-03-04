-- Test: calls.lua - Fixed-point operator (Y combinator)
-- From: calls.lua
-- What: Tests closures via a fixed-point combinator implementing non-recursive factorial

do
  local Z = function (le)
    local function a (f)
      return le(function (x) return f(f)(x) end)
    end
    return a(a)
  end

  local F = function (f)
    return function (n)
      if n == 0 then return 1
      else return n*f(n-1) end
    end
  end

  local fat = Z(F)
  assert(fat(0) == 1 and fat(4) == 24 and Z(F)(5)==5*Z(F)(4))
end
