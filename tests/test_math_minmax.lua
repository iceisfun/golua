-- test_math_minmax: math.min and math.max edge cases

-- no args should error
assert(not pcall(math.min), "min() no args")
assert(not pcall(math.max), "max() no args")

-- basic
assert(math.max(3, 5, 2, 1) == 5, "max(3,5,2,1)")
assert(math.min(3, 5, 2, 1) == 1, "min(3,5,2,1)")

-- non-number second arg should error
assert(not pcall(math.min, 1, true), "min(1, true) should error")
assert(not pcall(math.max, 1, "true"), "max(1, 'true') should error")
