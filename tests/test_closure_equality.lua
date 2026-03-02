-- Test: closure.lua - Closure equality
-- From: closure.lua
-- What: Tests that closures created in different iterations are distinct, but a function returned from same path is the same

do
  a = {}
  for i = 1, 5 do  a[i] = function (x) return i + a + _ENV end  end
  assert(a[3] ~= a[4] and a[4] ~= a[5])

  do
    local a = function (x)  return math.sin(_ENV[x])  end
    local function f() return a end
    assert(f() == f())
  end
end
