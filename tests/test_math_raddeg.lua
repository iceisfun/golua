-- test_math_raddeg: math.rad and math.deg

assert(math.rad(90) == math.pi / 2, "rad(90)")
assert(math.deg(math.pi) == 180, "deg(pi)")

-- no args should error
assert(not pcall(math.rad), "rad() no args")
assert(not pcall(math.deg), "deg() no args")

-- table arg should error
assert(not pcall(math.rad, {}), "rad({}) should error")
assert(not pcall(math.deg, {}), "deg({}) should error")
