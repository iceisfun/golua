-- test_math_random_ranges.lua
-- math.random must handle the full 64-bit integer range without overflow.

local min = math.mininteger -- -2^63
local max = math.maxinteger -- 2^63 - 1

-- Full 64-bit range [mininteger, maxinteger] must not panic
local ok, val = pcall(math.random, min, max)
assert(ok, "math.random(mininteger, maxinteger) panicked: " .. tostring(val))
assert(type(val) == "number", "expected number from full-range random")

-- Interval error still works
local ok2, err2 = pcall(math.random, 10, 1)
assert(not ok2, "math.random(10, 1) should error")

-- Reproducibility with seeded full-range random
math.randomseed(42, 0)
local r1 = math.random(min, max)
math.randomseed(42, 0)
local r2 = math.random(min, max)
assert(r1 == r2, "seeded full-range random should be reproducible")

-- Single-arg form with maxinteger
local ok3, val3 = pcall(math.random, max)
assert(ok3, "math.random(maxinteger) panicked: " .. tostring(val3))
assert(val3 >= 1 and val3 <= max, "math.random(maxinteger) out of range")

-- Equal bounds returns the bound itself
assert(math.random(5, 5) == 5, "math.random(5, 5) should return 5")
assert(math.random(min, min) == min, "math.random(mininteger, mininteger) should return mininteger")
assert(math.random(max, max) == max, "math.random(maxinteger, maxinteger) should return maxinteger")

-- Single-arg with 1 always returns 1
assert(math.random(1) == 1, "math.random(1) should return 1")
