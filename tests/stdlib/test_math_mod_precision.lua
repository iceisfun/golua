-- Test: math.lua - Modulo precision for large numbers
-- From: math.lua
-- What: Tests modulo by 3 for all powers of two up to the integer and float limits.

do
  do    -- precision of module for large numbers
    local i = 10
    while (1 << i) > 0 do
      assert((1 << i) % 3 == i % 2 + 1)
      i = i + 1
    end

    i = 10
    while 2^i < math.huge do
      assert(2^i % 3 == i % 2 + 1)
      i = i + 1
    end
  end
end
