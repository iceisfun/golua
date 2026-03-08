-- Test: trig functions produce negative NaN (matching C behavior)

local function is_neg_nan(x)
    return tostring(x) == "-nan"
end

assert(is_neg_nan(math.cos(0/0)), "cos(NaN) should be -nan")
assert(is_neg_nan(math.sin(math.huge)), "sin(huge) should be -nan")
assert(is_neg_nan(math.cos(math.huge)), "cos(huge) should be -nan")
assert(is_neg_nan(math.tan(math.huge)), "tan(huge) should be -nan")
assert(is_neg_nan(math.atan(0/0)), "atan(NaN) should be -nan")

print("OK")
