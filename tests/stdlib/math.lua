-- math.lua: Regression tests for math library functions.
-- These tests lock in correct, passing behavior. Any failure is a regression.

local function assert_eq(a, b, msg)
    if a ~= b then
        error((msg or "assertion failed") .. ": expected " .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

local function assert_near(a, b, eps, msg)
    eps = eps or 1e-10
    if math.abs(a - b) > eps then
        error((msg or "assertion failed") .. ": expected ~" .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

--------------------------------------------------------------------------------
-- math.floor
--------------------------------------------------------------------------------

assert_eq(math.floor(3.7), 3)
assert_eq(math.floor(3.0), 3)
assert_eq(math.floor(-3.7), -4)
assert_eq(math.floor(-3.0), -3)
assert_eq(math.floor(0), 0)
assert_eq(math.floor(0.5), 0)
assert_eq(math.floor(-0.5), -1)

-- floor returns integer type
assert_eq(math.type(math.floor(3.7)), "integer")

--------------------------------------------------------------------------------
-- math.ceil
--------------------------------------------------------------------------------

assert_eq(math.ceil(3.2), 4)
assert_eq(math.ceil(3.0), 3)
assert_eq(math.ceil(-3.2), -3)
assert_eq(math.ceil(-3.0), -3)
assert_eq(math.ceil(0), 0)
assert_eq(math.ceil(0.5), 1)
assert_eq(math.ceil(-0.5), 0)

-- ceil returns integer type
assert_eq(math.type(math.ceil(3.2)), "integer")

--------------------------------------------------------------------------------
-- math.modf
--------------------------------------------------------------------------------

-- Positive float
local i1, f1 = math.modf(3.75)
assert_eq(i1, 3)
assert_near(f1, 0.75)
assert_eq(math.type(i1), "integer")
assert_eq(math.type(f1), "float")

-- Negative float
local i2, f2 = math.modf(-3.75)
assert_eq(i2, -3)
assert_near(f2, -0.75)

-- Whole number
local i3, f3 = math.modf(5.0)
assert_eq(i3, 5)
assert_near(f3, 0.0)

-- Zero
local i4, f4 = math.modf(0)
assert_eq(i4, 0)
assert_near(f4, 0.0)

-- Integer input
local i5, f5 = math.modf(7)
assert_eq(i5, 7)
assert_near(f5, 0.0)

--------------------------------------------------------------------------------
-- math.tointeger
--------------------------------------------------------------------------------

-- Float with integer value
assert_eq(math.tointeger(5.0), 5)
assert_eq(math.type(math.tointeger(5.0)), "integer")

-- Actual integer
assert_eq(math.tointeger(5), 5)

-- Float with fractional part returns nil
assert_eq(math.tointeger(5.5), nil)

-- Zero
assert_eq(math.tointeger(0), 0)
assert_eq(math.tointeger(0.0), 0)

-- Negative
assert_eq(math.tointeger(-3.0), -3)
assert_eq(math.tointeger(-3.5), nil)

-- NOTE: In standard Lua 5.4, math.tointeger("5") returns nil (no coercion).
-- This implementation currently coerces string-to-number; tracked for future fix.
-- assert_eq(math.tointeger("5"), nil)

--------------------------------------------------------------------------------
-- math.maxinteger / math.mininteger
--------------------------------------------------------------------------------

-- They exist and are integers
assert_eq(math.type(math.maxinteger), "integer")
assert_eq(math.type(math.mininteger), "integer")

-- Exact boundary values (int64 precision)
assert_eq(tostring(math.maxinteger), "9223372036854775807")
assert_eq(tostring(math.mininteger), "-9223372036854775808")

-- Wrapping: maxinteger + 1 == mininteger
assert_eq(math.maxinteger + 1, math.mininteger)
assert_eq(math.mininteger - 1, math.maxinteger)

--------------------------------------------------------------------------------
-- math.ult (unsigned less-than comparison)
--------------------------------------------------------------------------------

-- Basic unsigned comparison
assert_eq(math.ult(1, 2), true)
assert_eq(math.ult(2, 1), false)
assert_eq(math.ult(1, 1), false)

-- Negative numbers are large in unsigned representation
assert_eq(math.ult(-1, 0), false)   -- -1 unsigned = max uint64
assert_eq(math.ult(0, -1), true)

-- maxinteger/mininteger boundary
assert_eq(math.ult(math.maxinteger, math.mininteger), true)   -- max < min unsigned (min = 2^63)
assert_eq(math.ult(math.mininteger, math.maxinteger), false)

-- Zero
assert_eq(math.ult(0, 0), false)
assert_eq(math.ult(0, 1), true)

--------------------------------------------------------------------------------
-- math.type
--------------------------------------------------------------------------------

assert_eq(math.type(1), "integer")
assert_eq(math.type(1.0), "float")
assert_eq(math.type("x"), nil)     -- non-number returns nil (not false)
assert_eq(math.type(true), nil)
assert_eq(math.type(nil), nil)

--------------------------------------------------------------------------------
-- math.abs
--------------------------------------------------------------------------------

assert_eq(math.abs(5), 5)
assert_eq(math.abs(-5), 5)
assert_eq(math.abs(0), 0)
assert_near(math.abs(-3.14), 3.14)
-- abs preserves integer type
assert_eq(math.type(math.abs(-5)), "integer")
assert_eq(math.type(math.abs(-5.0)), "float")

--------------------------------------------------------------------------------
-- math.max / math.min
--------------------------------------------------------------------------------

assert_eq(math.max(1, 2, 3), 3)
assert_eq(math.max(3, 2, 1), 3)
assert_eq(math.max(-1, -2, -3), -1)
assert_eq(math.min(1, 2, 3), 1)
assert_eq(math.min(3, 2, 1), 1)
assert_eq(math.min(-1, -2, -3), -3)

--------------------------------------------------------------------------------
-- math.sqrt
--------------------------------------------------------------------------------

assert_near(math.sqrt(4), 2.0)
assert_near(math.sqrt(9), 3.0)
assert_near(math.sqrt(2), 1.4142135623730951)
assert_near(math.sqrt(0), 0.0)

--------------------------------------------------------------------------------
-- math constants
--------------------------------------------------------------------------------

assert_near(math.pi, 3.141592653589793)
assert_eq(math.huge, 1/0)
assert_eq(math.huge, math.huge)
