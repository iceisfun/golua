-- Test: bitwise.lua - Constant folding for shifts
-- From: bitwise.lua
-- What: Tests that shift operations with maxinteger/mininteger amounts are properly folded to 0

do
  local code = string.format("return -1 >> %d", math.maxinteger)
  assert(load(code)() == 0)
  local code = string.format("return -1 >> %d", math.mininteger)
  assert(load(code)() == 0)
  local code = string.format("return -1 << %d", math.maxinteger)
  assert(load(code)() == 0)
  local code = string.format("return -1 << %d", math.mininteger)
  assert(load(code)() == 0)
end
