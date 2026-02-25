-- Bug #10: Float-to-string uses %.16g instead of Lua 5.4's %.14g

assert(tostring(1/3) == "0.33333333333333", "1/3 format wrong: " .. tostring(1/3))
assert(tostring(math.pi) == "3.1415926535898", "pi format wrong: " .. tostring(math.pi))
assert(tostring(2/3) == "0.66666666666667", "2/3 format wrong: " .. tostring(2/3))

-- Integer tostring should be unaffected
assert(tostring(42) == "42")
assert(tostring(0) == "0")
