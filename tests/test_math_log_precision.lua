-- Test: math.log with base should return correct values
-- Lua 5.4 returns exactly 3 for math.log(1000, 10).
-- GoLua returns 2.9999999999999996 due to floating-point precision.

-- math.log(8, 2) should be exactly 3
assert(math.log(8, 2) == 3, "math.log(8,2) should be exactly 3, got " .. string.format("%.17g", math.log(8, 2)))

-- math.log(1000, 10) should be exactly 3
assert(math.log(1000, 10) == 3, "math.log(1000,10) should be exactly 3, got " .. string.format("%.17g", math.log(1000, 10)))

-- math.log(100, 10) should be exactly 2
assert(math.log(100, 10) == 2, "math.log(100,10) should be exactly 2, got " .. string.format("%.17g", math.log(100, 10)))

-- math.log(256, 2) should be exactly 8
assert(math.log(256, 2) == 8, "math.log(256,2) should be exactly 8, got " .. string.format("%.17g", math.log(256, 2)))
