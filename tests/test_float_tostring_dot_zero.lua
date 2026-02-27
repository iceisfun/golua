-- Bug: tostring() on float values that format without a decimal point
-- should append ".0" to distinguish from integers.
-- lua5.4: tostring(2.9999999999999996) = "3.0"
-- golua:  tostring(2.9999999999999996) = "3"

-- Test 1: float that rounds to integer representation in %.14g
local s1 = tostring(2.9999999999999996)
assert(s1 == "3.0", "expected '3.0', got '" .. s1 .. "'")

-- Test 2: another near-integer float
local s2 = tostring(1.0000000000000002)
assert(s2 == "1.0", "expected '1.0', got '" .. s2 .. "'")

-- Test 3: large near-integer float
local s3 = tostring(9999999999999.998)
assert(s3 == "10000000000000.0", "expected '10000000000000.0', got '" .. s3 .. "'")

-- Test 4: 1.0 should be "1.0" (basic float)
local s4 = tostring(1.0)
assert(s4 == "1.0", "expected '1.0', got '" .. s4 .. "'")

-- Test 5: 0.0 should be "0.0"
local s5 = tostring(0.0)
assert(s5 == "0.0", "expected '0.0', got '" .. s5 .. "'")

-- Test 6: string concatenation should also have .0
local s6 = 1.0 .. ""
assert(s6 == "1.0", "concat: expected '1.0', got '" .. s6 .. "'")

print("PASS")
