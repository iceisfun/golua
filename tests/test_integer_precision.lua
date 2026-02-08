-- test_integer_precision.lua: Tests for int64 precision in the Value type.

local function assert_eq(a, b, msg)
    if a ~= b then
        error((msg or "assertion failed") .. ": expected " .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

--------------------------------------------------------------------------------
-- Large integer arithmetic
--------------------------------------------------------------------------------

-- Exact representation
local big = 2^53 + 1    -- 9007199254740993 (cannot be represented exactly in float64)
-- Note: 2^53 is computed as float, so big is float. Use integer operations instead.
local imax = math.maxinteger
local imin = math.mininteger

assert_eq(tostring(imax), "9223372036854775807", "maxinteger tostring")
assert_eq(tostring(imin), "-9223372036854775808", "mininteger tostring")

-- Wrapping arithmetic
assert_eq(imax + 1, imin, "maxinteger + 1 == mininteger (wrapping)")
assert_eq(imin - 1, imax, "mininteger - 1 == maxinteger (wrapping)")
assert_eq(imin + imax, -1, "mininteger + maxinteger == -1")

-- Multiplication
assert_eq(1000000 * 1000000, 1000000000000, "million squared")
assert_eq(1000000000 * 1000000000, 1000000000000000000, "billion squared")

--------------------------------------------------------------------------------
-- Integer for loop
--------------------------------------------------------------------------------

-- Basic integer for loop
local sum = 0
for i = 1, 10 do
    sum = sum + i
end
assert_eq(sum, 55, "sum 1..10")

-- Integer for loop preserves integer type
for i = 1, 3 do
    assert_eq(math.type(i), "integer", "for loop var should be integer")
end

-- Negative step
local vals = {}
for i = 5, 1, -1 do
    vals[#vals + 1] = i
end
assert_eq(#vals, 5, "negative step count")
assert_eq(vals[1], 5, "negative step first")
assert_eq(vals[5], 1, "negative step last")

-- Float for loop should produce floats
for i = 1.0, 3.0 do
    assert_eq(math.type(i), "float", "float for loop var should be float")
end

--------------------------------------------------------------------------------
-- Integer division and modulo
--------------------------------------------------------------------------------

assert_eq(7 // 2, 3, "7 // 2")
assert_eq(-7 // 2, -4, "Lua floor division: -7 // 2 == -4")
assert_eq(7 // -2, -4, "7 // -2 == -4")
assert_eq(-7 // -2, 3, "-7 // -2 == 3")

assert_eq(7 % 2, 1, "7 %% 2")
assert_eq(-7 % 2, 1, "Lua mod: -7 %% 2 == 1")
assert_eq(7 % -2, -1, "7 %% -2 == -1")
assert_eq(-7 % -2, -1, "-7 %% -2 == -1")

-- Integer division/mod should return integers when both operands are integers
assert_eq(math.type(7 // 2), "integer", "int // int should be integer")
assert_eq(math.type(7 % 2), "integer", "int %% int should be integer")

-- Division always returns float
assert_eq(math.type(7 / 2), "float", "/ always returns float")

--------------------------------------------------------------------------------
-- Bitwise operations on large values
--------------------------------------------------------------------------------

assert_eq(imax & imax, imax, "maxinteger & maxinteger")
assert_eq(imax | 0, imax, "maxinteger | 0")
assert_eq(imax ~ 0, imax, "maxinteger ~ 0")
assert_eq(~0, -1, "~0 == -1")
assert_eq(~imax, imin, "~maxinteger == mininteger")

-- Shift operations
assert_eq(1 << 62, 4611686018427387904, "1 << 62")
assert_eq(1 << 63, imin, "1 << 63 == mininteger (wrapping)")

--------------------------------------------------------------------------------
-- Float vs int comparison edge cases
--------------------------------------------------------------------------------

-- maxinteger should NOT equal maxinteger + 0.0 (float loses precision)
assert_eq(imax == imax + 0.0, false, "maxinteger ~= maxinteger + 0.0")

-- mininteger SHOULD equal mininteger + 0.0 (float64 can represent -2^63 exactly)
assert_eq(imin == imin + 0.0, true, "mininteger == mininteger + 0.0")

-- 1 == 1.0 (standard case)
assert_eq(1 == 1.0, true, "1 == 1.0")

-- Ordering across types
assert_eq(1 < 1.5, true, "int < float")
assert_eq(1.5 < 2, true, "float < int")
assert_eq(1 <= 1.0, true, "int <= float equal")
assert_eq(1.0 <= 1, true, "float <= int equal")

--------------------------------------------------------------------------------
-- tonumber with large integers
--------------------------------------------------------------------------------

local n = tonumber("9223372036854775807")
assert_eq(n, imax, "tonumber large int string")
assert_eq(math.type(n), "integer", "tonumber large int should be integer type")

local n2 = tonumber("-9223372036854775808")
assert_eq(n2, imin, "tonumber large negative int string")

--------------------------------------------------------------------------------
-- string.format with large integers
--------------------------------------------------------------------------------

assert_eq(string.format("%d", imax), "9223372036854775807", "format maxinteger")
assert_eq(string.format("%d", imin), "-9223372036854775808", "format mininteger")
