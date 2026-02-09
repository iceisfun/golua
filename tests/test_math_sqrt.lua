-- test_math_sqrt: math.sqrt edge cases

-- no args should error
assert(not pcall(math.sqrt), "sqrt() no args")

-- table arg should error
assert(not pcall(math.sqrt, {}), "sqrt({}) should error")

-- known values
assert(math.floor(1000*math.sqrt(1)) == 1000, "sqrt(1)")
assert(math.floor(1000*math.sqrt(2)) == 1414, "sqrt(2)")
assert(math.floor(1000*math.sqrt(3)) == 1732, "sqrt(3)")
assert(math.floor(1000*math.sqrt(4)) == 2000, "sqrt(4)")
assert(math.floor(1000*math.sqrt(5)) == 2236, "sqrt(5)")
