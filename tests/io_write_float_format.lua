-- Test: io.write/file:write uses same format as tostring() for floats
-- Lua 5.5: io.write uses the shortest round-trip representation,
-- including ".0" suffix for integer-valued floats.

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

-- Integer-valued floats: ".0" suffix (same as tostring)
assert(capture_write(42.0) == "42.0", "42.0 -> '" .. capture_write(42.0) .. "'")
assert(capture_write(1.0) == "1.0", "1.0 -> '" .. capture_write(1.0) .. "'")
assert(capture_write(0.0) == "0.0", "0.0 -> '" .. capture_write(0.0) .. "'")
assert(capture_write(100.0) == "100.0", "100.0 -> '" .. capture_write(100.0) .. "'")
assert(capture_write(1e10) == "10000000000.0", "1e10 -> '" .. capture_write(1e10) .. "'")

-- Negative zero
assert(capture_write(-0.0) == "-0.0", "-0.0 -> '" .. capture_write(-0.0) .. "'")

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

-- Verify tostring matches io.write for floats (Lua 5.5 unification)
assert(tostring(42.0) == "42.0", "tostring(42.0) should have .0")
assert(tostring(1.0) == "1.0", "tostring(1.0) should have .0")
assert(tostring(0.0) == "0.0", "tostring(0.0) should have .0")
assert(capture_write(42.0) == tostring(42.0), "io.write should match tostring for floats")
