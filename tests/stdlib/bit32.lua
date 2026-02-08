-- bit32.lua: Tests for the bit32 compatibility library.
-- All operations work on unsigned 32-bit integers.

local function assert_eq(a, b, msg)
    if a ~= b then
        error((msg or "assertion failed") .. ": expected " .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

--------------------------------------------------------------------------------
-- bit32.band
--------------------------------------------------------------------------------

assert_eq(bit32.band(0xFF, 0x0F), 0x0F, "band basic")
assert_eq(bit32.band(0xFFFFFFFF, 0x12345678), 0x12345678, "band with all ones")
assert_eq(bit32.band(0xFF00, 0x00FF), 0, "band disjoint")
assert_eq(bit32.band(0xFF, 0xFF, 0x0F), 0x0F, "band variadic")
-- No args returns all ones (identity element for AND)
assert_eq(bit32.band(), 0xFFFFFFFF, "band no args")

--------------------------------------------------------------------------------
-- bit32.bor
--------------------------------------------------------------------------------

assert_eq(bit32.bor(0xFF00, 0x00FF), 0xFFFF, "bor basic")
assert_eq(bit32.bor(0, 0), 0, "bor zeros")
assert_eq(bit32.bor(1, 2, 4, 8), 15, "bor variadic")
-- No args returns 0 (identity element for OR)
assert_eq(bit32.bor(), 0, "bor no args")

--------------------------------------------------------------------------------
-- bit32.bxor
--------------------------------------------------------------------------------

assert_eq(bit32.bxor(0xFF, 0xFF), 0, "bxor same")
assert_eq(bit32.bxor(0xFF, 0x00), 0xFF, "bxor with zero")
assert_eq(bit32.bxor(0xAA, 0x55), 0xFF, "bxor complementary")
assert_eq(bit32.bxor(1, 2, 3), 0, "bxor variadic: 1^2^3 = 0")
assert_eq(bit32.bxor(), 0, "bxor no args")

--------------------------------------------------------------------------------
-- bit32.bnot
--------------------------------------------------------------------------------

assert_eq(bit32.bnot(0), 0xFFFFFFFF, "bnot 0")
assert_eq(bit32.bnot(0xFFFFFFFF), 0, "bnot all ones")
assert_eq(bit32.bnot(0x0F0F0F0F), 0xF0F0F0F0, "bnot pattern")

--------------------------------------------------------------------------------
-- bit32.btest
--------------------------------------------------------------------------------

assert_eq(bit32.btest(0xFF, 0x0F), true, "btest overlap")
assert_eq(bit32.btest(0xFF00, 0x00FF), false, "btest no overlap")
assert_eq(bit32.btest(0xFF, 0xFF, 0x01), true, "btest variadic")

--------------------------------------------------------------------------------
-- Input truncation (values > 32 bits get masked)
--------------------------------------------------------------------------------

-- 0x1FFFFFFFF should become 0xFFFFFFFF after uint32 truncation
assert_eq(bit32.band(0x1FFFFFFFF, 0xFFFFFFFF), 0xFFFFFFFF, "truncation")

--------------------------------------------------------------------------------
-- bit32.lshift
--------------------------------------------------------------------------------

assert_eq(bit32.lshift(1, 0), 1, "lshift by 0")
assert_eq(bit32.lshift(1, 1), 2, "lshift by 1")
assert_eq(bit32.lshift(1, 31), 2147483648, "lshift to MSB")
assert_eq(bit32.lshift(1, 32), 0, "lshift out of bounds")
assert_eq(bit32.lshift(0xFF, 8), 0xFF00, "lshift byte")
-- Negative displacement = right shift
assert_eq(bit32.lshift(0xFF00, -8), 0xFF, "lshift negative disp")

--------------------------------------------------------------------------------
-- bit32.rshift
--------------------------------------------------------------------------------

assert_eq(bit32.rshift(2, 1), 1, "rshift by 1")
assert_eq(bit32.rshift(0xFF00, 8), 0xFF, "rshift byte")
assert_eq(bit32.rshift(0x80000000, 31), 1, "rshift MSB")
assert_eq(bit32.rshift(0x80000000, 32), 0, "rshift out of bounds")
-- Logical: does NOT preserve sign bit
assert_eq(bit32.rshift(0x80000000, 1), 0x40000000, "rshift logical (no sign extension)")
-- Negative displacement = left shift
assert_eq(bit32.rshift(1, -8), 0x100, "rshift negative disp")

--------------------------------------------------------------------------------
-- bit32.arshift (arithmetic right shift)
--------------------------------------------------------------------------------

-- When MSB is 0, arshift == rshift
assert_eq(bit32.arshift(0x7FFFFFFF, 1), 0x3FFFFFFF, "arshift positive")

-- When MSB is 1 (sign bit set), arshift preserves sign
local neg = 0x80000000
assert_eq(bit32.arshift(neg, 1), 0xC0000000, "arshift sign extension")
assert_eq(bit32.arshift(neg, 4), 0xF8000000, "arshift sign extension 4")
assert_eq(bit32.arshift(neg, 31), 0xFFFFFFFF, "arshift max shift signed")

-- Shift >= 32 fills with sign bit
assert_eq(bit32.arshift(0x80000000, 32), 0xFFFFFFFF, "arshift >= 32 negative")
assert_eq(bit32.arshift(0x7FFFFFFF, 32), 0, "arshift >= 32 positive")
assert_eq(bit32.arshift(0x80000000, 100), 0xFFFFFFFF, "arshift >> 32 negative")

-- Shift by 0
assert_eq(bit32.arshift(0xDEADBEEF, 0), 0xDEADBEEF, "arshift by 0")

-- Negative displacement = left shift (same as lshift)
assert_eq(bit32.arshift(1, -4), 16, "arshift negative disp")
assert_eq(bit32.arshift(1, -32), 0, "arshift negative disp >= 32")

-- 0xFFFFFFFF arshift should stay 0xFFFFFFFF
assert_eq(bit32.arshift(0xFFFFFFFF, 1), 0xFFFFFFFF, "arshift all ones")
assert_eq(bit32.arshift(0xFFFFFFFF, 16), 0xFFFFFFFF, "arshift all ones by 16")

--------------------------------------------------------------------------------
-- bit32.extract
--------------------------------------------------------------------------------

-- Extract 4 bits from position 4: 0xAC (10101100) -> bits 7..4 = 1010 = 10
assert_eq(bit32.extract(0xAC, 4, 4), 10, "extract nibble")
-- Extract single bit (default width=1)
assert_eq(bit32.extract(0x05, 0), 1, "extract bit 0")
assert_eq(bit32.extract(0x05, 1), 0, "extract bit 1")
assert_eq(bit32.extract(0x05, 2), 1, "extract bit 2")
-- Extract full 32 bits
assert_eq(bit32.extract(0xDEADBEEF, 0, 32), 0xDEADBEEF, "extract all 32")
-- Extract high byte
assert_eq(bit32.extract(0xABCD1234, 24, 8), 0xAB, "extract high byte")

--------------------------------------------------------------------------------
-- bit32.replace
--------------------------------------------------------------------------------

-- Replace low nibble
assert_eq(bit32.replace(0xF0, 0x0A, 0, 4), 0xFA, "replace low nibble")
-- Replace single bit
assert_eq(bit32.replace(0x00, 1, 7), 0x80, "replace single bit")
-- Replace high byte
assert_eq(bit32.replace(0x12345678, 0xFF, 24, 8), 0xFF345678, "replace high byte")

--------------------------------------------------------------------------------
-- bit32.lrotate
--------------------------------------------------------------------------------

assert_eq(bit32.lrotate(1, 1), 2, "lrotate by 1")
assert_eq(bit32.lrotate(0x80000000, 1), 1, "lrotate MSB wraps")
assert_eq(bit32.lrotate(0x12345678, 32), 0x12345678, "lrotate full rotation")
assert_eq(bit32.lrotate(0x12345678, 0), 0x12345678, "lrotate by 0")
-- Negative = right rotate
assert_eq(bit32.lrotate(1, -1), 0x80000000, "lrotate negative")

--------------------------------------------------------------------------------
-- bit32.rrotate
--------------------------------------------------------------------------------

assert_eq(bit32.rrotate(1, 1), 0x80000000, "rrotate by 1")
assert_eq(bit32.rrotate(0x80000000, 1), 0x40000000, "rrotate MSB")
assert_eq(bit32.rrotate(0x12345678, 32), 0x12345678, "rrotate full rotation")
-- Negative = left rotate
assert_eq(bit32.rrotate(2, -1), 4, "rrotate negative")
