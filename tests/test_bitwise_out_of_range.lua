-- Test: bitwise.lua - Out of range and embedded zeros
-- From: bitwise.lua
-- What: Tests that out-of-range float strings and strings with embedded zeros fail for bitwise ops

do
  assert(not pcall(function () return "0xffffffffffffffff.0" | 0 end))
  assert(not pcall(function () return "0xffffffffffffffff\0" | 0 end))
end
