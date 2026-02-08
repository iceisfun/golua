-- binary_literals.lua: Tests for 0b/0B binary integer literals.

local function assert_eq(a, b, msg)
    if a ~= b then
        error((msg or "assertion failed") .. ": expected " .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

--------------------------------------------------------------------------------
-- Basic values
--------------------------------------------------------------------------------

assert_eq(0b0, 0, "0b0")
assert_eq(0b1, 1, "0b1")
assert_eq(0b10, 2, "0b10")
assert_eq(0b11, 3, "0b11")
assert_eq(0b1010, 10, "0b1010")
assert_eq(0b1111, 15, "0b1111")
assert_eq(0b11111111, 255, "0b11111111")
assert_eq(0B1010, 10, "0B1010 uppercase prefix")

--------------------------------------------------------------------------------
-- Arithmetic with binary literals
--------------------------------------------------------------------------------

assert_eq(0b1010 + 0b0101, 15, "binary addition")
assert_eq(0b1000 - 0b0001, 7, "binary subtraction")
assert_eq(0b10 * 0b11, 6, "binary multiplication")
assert_eq(0b1010 // 0b10, 5, "binary floor division")

--------------------------------------------------------------------------------
-- Bitwise operations with binary literals
--------------------------------------------------------------------------------

assert_eq(0b1 | 0b10, 3, "binary bitwise or")
assert_eq(0b1111 & 0b1010, 10, "binary bitwise and")
assert_eq(0b1111 ~ 0b1010, 5, "binary bitwise xor")
assert_eq(0b1 << 3, 8, "binary left shift")
assert_eq(0b1000 >> 2, 2, "binary right shift")

--------------------------------------------------------------------------------
-- Comparisons with decimal and hex
--------------------------------------------------------------------------------

assert_eq(0b1010, 10, "binary equals decimal")
assert_eq(0b1010, 0xA, "binary equals hex")
assert(0b1010 == 10, "binary == decimal")
assert(0b1010 == 0xa, "binary == hex")
assert(0b1 < 0b10, "binary less than")
assert(0b10 > 0b1, "binary greater than")

--------------------------------------------------------------------------------
-- Type check: binary literals are integers
--------------------------------------------------------------------------------

assert_eq(type(0b1010), "number", "binary is number type")
assert_eq(math.type(0b1010), "integer", "binary is integer subtype")

--------------------------------------------------------------------------------
-- 64-bit boundary
--------------------------------------------------------------------------------

assert_eq(0b1111111111111111111111111111111111111111111111111111111111111111,
          -1, "all 64 bits set wraps to -1")

--------------------------------------------------------------------------------
-- Mixed expressions
--------------------------------------------------------------------------------

local x = 0b1010
local y = 0b0101
assert_eq(x + y, 15, "binary locals sum")
assert_eq(x | y, 15, "binary locals bitwise or")
assert_eq(x & y, 0, "binary locals bitwise and")

--------------------------------------------------------------------------------
-- Invalid binary literals must fail to compile
--------------------------------------------------------------------------------

local fn1, err1 = load("return 0b102")
assert(fn1 == nil, "0b102 must fail to compile")

local fn2, err2 = load("return 0b")
assert(fn2 == nil, "0b alone must fail to compile")
