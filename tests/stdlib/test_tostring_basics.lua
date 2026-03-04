-- Test: tostring basics
-- From: strings.lua
-- What: Tests tostring for nil, numbers, tables, functions, booleans, and integer edge cases including 32-bit and 64-bit boundaries.

do
assert(type(tostring(nil)) == 'string')
assert(type(tostring(12)) == 'string')
assert(string.find(tostring{}, 'table:'))
assert(string.find(tostring(print), 'function:'))
assert(#tostring('\0') == 1)
assert(tostring(true) == "true")
assert(tostring(false) == "false")
assert(tostring(-1203) == "-1203")
assert(tostring(1203.125) == "1203.125")
assert(tostring(-0.5) == "-0.5")
assert(tostring(-32767) == "-32767")
if math.tointeger(2147483647) then   -- no overflow? (32 bits)
  assert(tostring(-2147483647) == "-2147483647")
end
if math.tointeger(4611686018427387904) then   -- no overflow? (64 bits)
  assert(tostring(4611686018427387904) == "4611686018427387904")
  assert(tostring(-4611686018427387904) == "-4611686018427387904")
end
end
