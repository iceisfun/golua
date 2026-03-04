-- Test: bitwise.lua - Basic bitwise operators
-- From: bitwise.lua
-- What: Tests AND, OR, XOR, NOT, and shift operators with various values

do
  local a, b, c, d
  a = 0xFFFFFFFFFFFFFFFF
  assert(a == -1 and a & -1 == a and a & 35 == 35)
  a = 0xF0F0F0F0F0F0F0F0
  assert(a | -1 == -1)
  assert(a ~ a == 0 and a ~ 0 == a and a ~ ~a == -1)
  assert(a >> 4 == ~a)
  a = 0xF0; b = 0xCC; c = 0xAA; d = 0xFD
  assert(a | b ~ c & d == 0xF4)
end
