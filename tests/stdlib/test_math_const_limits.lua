-- Test: math.lua - Constant limits (2^23)
-- From: math.lua
-- What: Tests addition around 2^23 boundary to verify constant folding precision.

do
  assert(8388609 + -8388609 == 0)
  assert(8388608 + -8388608 == 0)
  assert(8388607 + -8388607 == 0)
end
