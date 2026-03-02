-- Test: math.lua - Integer and float modulo consistency
-- From: math.lua
-- What: Tests that integer modulo and float modulo produce consistent results for small values and powers of two.

do
  for i = -10, 10 do
    for j = -10, 10 do
      if j ~= 0 then
        assert((i + 0.0) % j == i % j)
      end
    end
  end

  for i = 0, 10 do
    for j = -10, 10 do
      if j ~= 0 then
        assert((2^i) % j == (1 << i) % j)
      end
    end
  end
end
