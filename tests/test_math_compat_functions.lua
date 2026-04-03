-- Lua 5.5: deprecated math functions removed (atan2, log10, pow, cosh, sinh, tanh)
-- frexp and ldexp are still present

assert(math.atan2 == nil, "math.atan2 should be removed in 5.5")
assert(math.log10 == nil, "math.log10 should be removed in 5.5")
assert(math.pow == nil, "math.pow should be removed in 5.5")
assert(math.cosh == nil, "math.cosh should be removed in 5.5")
assert(math.sinh == nil, "math.sinh should be removed in 5.5")
assert(math.tanh == nil, "math.tanh should be removed in 5.5")

-- frexp and ldexp still exist
assert(type(math.frexp) == "function")
assert(type(math.ldexp) == "function")

assert(math.ldexp(0.5, 4) == 8.0)

do
  local m, e = math.frexp(8)
  assert(math.type(m) == "float")
  assert(math.type(e) == "integer")
  assert(m == 0.5 and e == 4)
end

do
  local ok, err = pcall(math.ldexp, 1, 0/0)
  assert(not ok)
  assert(type(err) == "string" and string.find(err, "number has no integer representation", 1, true), tostring(err))
end
