-- Float-to-string uses Lua 5.5 shortest round-trip representation

assert(tostring(1/3) == "0.33333333333333331", "1/3 format wrong: " .. tostring(1/3))
assert(tostring(math.pi) == "3.1415926535897931", "pi format wrong: " .. tostring(math.pi))
assert(tostring(2/3) == "0.66666666666666663", "2/3 format wrong: " .. tostring(2/3))

-- Integer tostring should be unaffected
assert(tostring(42) == "42")
assert(tostring(0) == "0")
