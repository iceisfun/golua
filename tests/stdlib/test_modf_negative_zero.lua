-- Test that math.modf normalizes negative zero in fractional part
-- Go's math.Modf(-3.0) returns (-3.0, -0.0) but Lua 5.4 returns (-3, 0.0)

-- Fractional part of negative integer should be positive zero
local i, f = math.modf(-3)
assert(i == -3)
assert(f == 0.0)
assert(1/f == math.huge, "fractional part should be positive zero")

-- Fractional part of positive integer should be positive zero
local i2, f2 = math.modf(3)
assert(i2 == 3)
assert(f2 == 0.0)
assert(1/f2 == math.huge, "fractional part should be positive zero")

-- Negative zero input: both parts become positive zero (integer type for i)
local i3, f3 = math.modf(-0.0)
assert(i3 == 0)
assert(math.type(i3) == "integer")
assert(f3 == 0.0)
assert(1/f3 == math.huge, "fractional part of -0.0 should be positive zero")

-- Non-zero fractional part preserves sign
local i4, f4 = math.modf(-3.5)
assert(i4 == -3)
assert(f4 == -0.5)

