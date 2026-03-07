-- Test that special float values use Lua conventions when converted to string
-- Lua 5.4 uses: inf, -inf, -nan (not Go's +Inf, -Inf, NaN)

-- Positive infinity
assert(math.huge .. "" == "inf", "expected 'inf', got '" .. (math.huge .. "") .. "'")
assert(tostring(math.huge) == "inf")

-- Negative infinity
assert(-math.huge .. "" == "-inf", "expected '-inf', got '" .. (-math.huge .. "") .. "'")
assert(tostring(-math.huge) == "-inf")

-- NaN
assert((0/0) .. "" == "-nan", "expected '-nan', got '" .. ((0/0) .. "") .. "'")
assert(tostring(0/0) == "-nan")

-- Verify these are still floats
assert(math.type(math.huge) == "float")
assert(math.type(0/0) == "float")

-- Verify normal floats still format correctly
assert(tostring(1.5) == "1.5")
assert(tostring(1.0) == "1.0")

