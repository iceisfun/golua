-- Test: math.lua - tonumber edge cases with scientific notation
-- From: math.lua
-- What: Tests tonumber with whitespace-padded scientific notation strings.

do
  assert(tonumber(' 1.3e-2 ') == 1.3e-2)
  assert(tonumber(' -1.00000000000001 ') == -1.00000000000001)
end
