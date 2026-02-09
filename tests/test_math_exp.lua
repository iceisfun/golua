-- test_math_exp: math.exp edge cases

assert(math.exp(0) == 1.0, "exp(0) should be 1.0")

-- no args should error
assert(not pcall(math.exp), "exp() no args")

-- table arg should error
assert(not pcall(math.exp, {}), "exp({}) should error")
