-- BUG: tonumber fails on hex floats without integer part
-- In Lua 5.4, hex float literals like "0x.1" are valid and parse
-- correctly. GoLua returns nil for these.

-- Hex floats without integer part
assert(tonumber("0x.1") == 0.0625,
    "tonumber('0x.1') should be 0.0625, got: " .. tostring(tonumber("0x.1")))
assert(tonumber("0x.8") == 0.5,
    "tonumber('0x.8') should be 0.5, got: " .. tostring(tonumber("0x.8")))
assert(tonumber("0x.F") == 0.9375,
    "tonumber('0x.F') should be 0.9375, got: " .. tostring(tonumber("0x.F")))

-- Hex floats with both integer and fractional parts
assert(tonumber("0xA.8") == 10.5,
    "tonumber('0xA.8') should be 10.5, got: " .. tostring(tonumber("0xA.8")))
assert(tonumber("0x1.0") == 1.0,
    "tonumber('0x1.0') should be 1.0, got: " .. tostring(tonumber("0x1.0")))

-- Hex float with fractional part and exponent
assert(tonumber("0x.1p4") == 1.0,
    "tonumber('0x.1p4') should be 1.0, got: " .. tostring(tonumber("0x.1p4")))
assert(tonumber("0x.Fp0") == 0.9375,
    "tonumber('0x.Fp0') should be 0.9375, got: " .. tostring(tonumber("0x.Fp0")))

-- Hex floats with both parts and exponent
assert(tonumber("0xF.Fp4") == 255.0,
    "tonumber('0xF.Fp4') should be 255.0, got: " .. tostring(tonumber("0xF.Fp4")))

-- Arithmetic coercion of hex float strings should also work
assert("0x.1" + 0 == 0.0625,
    "'0x.1' + 0 should be 0.0625")
assert("0xA.8" + 0 == 10.5,
    "'0xA.8' + 0 should be 10.5")

-- Hex floats that already work should still work
assert(tonumber("0x1p10") == 1024.0, "0x1p10 should work")
assert(tonumber("0x1.8p1") == 3.0, "0x1.8p1 should work")
assert(tonumber("0xFp0") == 15.0, "0xFp0 should work")
