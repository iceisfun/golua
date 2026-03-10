assert(type(math.atan2) == "function")
assert(type(math.log10) == "function")
assert(type(math.pow) == "function")
assert(type(math.frexp) == "function")
assert(type(math.ldexp) == "function")
assert(type(math.cosh) == "function")
assert(type(math.sinh) == "function")
assert(type(math.tanh) == "function")
assert(math.atan2 == math.atan)

assert(math.log10(1000) == 3.0)
assert(math.pow(2, 3) == 8.0)
assert(math.ldexp(0.5, 4) == 8.0)
assert(math.cosh(0) == 1.0)
assert(math.sinh(0) == 0.0)
assert(math.tanh(0) == 0.0)

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
