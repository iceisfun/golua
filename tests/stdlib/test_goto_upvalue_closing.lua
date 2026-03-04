-- Test: goto.lua - Closing of upvalues with goto
-- From: goto.lua
-- What: Tests that upvalues are correctly closed and shared across closures around goto labels

do
  local debug = require 'debug'

  local function foo ()
    local t = {}
    do
    local i = 1
    local a, b, c, d
    t[1] = function () return a, b, c, d end
    ::l1::
    local b
    do
      local c
      t[#t + 1] = function () return a, b, c, d end
      if i > 2 then goto l2 end
      do
        local d
        t[#t + 1] = function () return a, b, c, d end
        i = i + 1
        local a
        goto l1
      end
    end
    end
    ::l2:: return t
  end

  local a = foo()
  assert(#a == 6)

  -- all closures share the same 'a' upvalue (first upvalue)
  for i = 2, 6 do
    assert(debug.upvalueid(a[1], 1) == debug.upvalueid(a[i], 1))
  end

  -- closures get different 'b' and 'c' upvalues from t[1]
  for i = 2, 6 do
    assert(debug.upvalueid(a[1], 2) ~= debug.upvalueid(a[i], 2))
    assert(debug.upvalueid(a[1], 3) ~= debug.upvalueid(a[i], 3))
  end

  -- pairs created together share 'b' and 'c' but differ from the next pair
  for i = 3, 5, 2 do
    assert(debug.upvalueid(a[i], 2) == debug.upvalueid(a[i - 1], 2))
    assert(debug.upvalueid(a[i], 3) == debug.upvalueid(a[i - 1], 3))
    assert(debug.upvalueid(a[i], 2) ~= debug.upvalueid(a[i + 1], 2))
    assert(debug.upvalueid(a[i], 3) ~= debug.upvalueid(a[i + 1], 3))
  end

  -- even-indexed closures share outer 'd' with t[1]
  for i = 2, 6, 2 do
    assert(debug.upvalueid(a[1], 4) == debug.upvalueid(a[i], 4))
  end

  -- odd-indexed closures (except t[1]) have unique 'd' upvalues
  for i = 3, 5, 2 do
    for j = 1, 6 do
      assert((debug.upvalueid(a[i], 4) == debug.upvalueid(a[j], 4))
        == (i == j))
    end
  end
end
