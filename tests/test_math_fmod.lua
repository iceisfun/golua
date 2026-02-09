-- test_math_fmod: math.fmod edge cases

-- basic
assert(math.fmod(5, 2) == 1, "fmod(5, 2)")

-- negative with same magnitude
assert(math.fmod(-6, -6) == 0, "fmod(-6, -6) integer")
assert(math.fmod(-6.0, -6) == 0, "fmod(-6.0, -6) float")

-- no args should error
assert(not pcall(math.fmod), "fmod() no args should error")

-- non-numeric string arg should error
assert(not pcall(math.fmod, "2", "a"), "fmod('2', 'a') should error")

-- division by zero should error
assert(not pcall(math.fmod, 3, 0), "fmod(3, 0) should error")
