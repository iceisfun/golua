-- test_math_abs: math.abs edge cases

-- basic integer
assert(math.abs(12) == 12, "abs(12)")
assert(math.abs(-12) == 12, "abs(-12)")

-- float
assert(math.abs(-0.5) == 0.5, "abs(-0.5)")

-- string coercion should work
local r = math.abs("-123")
assert(r == 123, string.format("abs('-123') expected 123, got %s", tostring(r)))

-- no args should error
assert(not pcall(math.abs), "abs() with no args should error")

-- table arg should error
assert(not pcall(math.abs, {}), "abs({}) should error")
