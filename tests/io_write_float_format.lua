-- Test: io.write/file:write uses %.14g for floats (no ".0" suffix)
-- Unlike tostring() which appends ".0" to integer-valued floats,
-- io.write uses C's fprintf with %.14g directly.

-- Capture io.write output via temp file
local function capture_write(...)
    local f = io.open("/tmp/golua_test_iowrite.txt", "w")
    f:write(...)
    f:close()
    local r = io.open("/tmp/golua_test_iowrite.txt", "r")
    local s = r:read("*a")
    r:close()
    os.remove("/tmp/golua_test_iowrite.txt")
    return s
end

-- Integer-valued floats: no ".0" suffix
assert(capture_write(42.0) == "42", "42.0 -> '42'")
assert(capture_write(1.0) == "1", "1.0 -> '1'")
assert(capture_write(0.0) == "0", "0.0 -> '0'")
assert(capture_write(100.0) == "100", "100.0 -> '100'")
assert(capture_write(1e10) == "10000000000", "1e10")

-- Negative zero
assert(capture_write(-0.0) == "-0", "-0.0 -> '-0'")

-- Non-integer floats: unchanged
assert(capture_write(1.5) == "1.5", "1.5 -> '1.5'")
assert(capture_write(0.1) == "0.1", "0.1 -> '0.1'")

-- Integers: unchanged
assert(capture_write(42) == "42", "42 -> '42'")
assert(capture_write(0) == "0", "0 -> '0'")

-- Special values
assert(capture_write(1/0) == "inf", "inf")
assert(capture_write(-1/0) == "-inf", "-inf")
assert(capture_write(0/0) == "-nan" or capture_write(0/0) == "nan", "nan")

-- Scientific notation (already has 'e', no ".0" issue)
assert(capture_write(1e100) == "1e+100", "1e100")

-- Verify tostring DOES have ".0" (to confirm the distinction)
assert(tostring(42.0) == "42.0", "tostring(42.0) should have .0")
assert(tostring(1.0) == "1.0", "tostring(1.0) should have .0")
assert(tostring(0.0) == "0.0", "tostring(0.0) should have .0")
