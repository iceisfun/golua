-- Bug: tonumber(s, base) rejects values > INT64_MAX instead of wrapping
-- them to signed int64 as Lua 5.4 does (unsigned wrapping).

-- Test 1: 0xffffffffffffffff should wrap to -1
local r1 = tonumber("ffffffffffffffff", 16)
assert(r1 == -1, "0xffffffffffffffff should be -1, got " .. tostring(r1))

-- Test 2: 0x8000000000000000 should be math.mininteger
local r2 = tonumber("8000000000000000", 16)
assert(r2 == math.mininteger,
  "0x8000000000000000 should be mininteger, got " .. tostring(r2))

-- Test 3: decimal 18446744073709551615 (2^64-1) should wrap to -1
local r3 = tonumber("18446744073709551615", 10)
assert(r3 == -1, "2^64-1 decimal should be -1, got " .. tostring(r3))

-- Test 4: values within INT64 range should still work normally
local r4 = tonumber("7fffffffffffffff", 16)
assert(r4 == math.maxinteger,
  "0x7fffffffffffffff should be maxinteger, got " .. tostring(r4))

-- Test 5: octal max unsigned
local r5 = tonumber("1777777777777777777777", 8)
assert(r5 == -1, "octal max unsigned should be -1, got " .. tostring(r5))

