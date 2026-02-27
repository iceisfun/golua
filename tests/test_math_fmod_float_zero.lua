-- Bug: math.fmod(float, 0.0) should return NaN, not error.
-- Only integer%0 should error. Float division by zero returns NaN per IEEE 754.

-- Test 1: math.fmod(1.0, 0.0) should return NaN
local r1 = math.fmod(1.0, 0.0)
assert(r1 ~= r1, "math.fmod(1.0, 0.0) should be NaN, got " .. tostring(r1))

-- Test 2: math.fmod(1.0, 0) with mixed types should return NaN
local r2 = math.fmod(1.0, 0)
assert(r2 ~= r2, "math.fmod(1.0, 0) should be NaN, got " .. tostring(r2))

-- Test 3: math.fmod(1, 0.0) with mixed types should return NaN
local r3 = math.fmod(1, 0.0)
assert(r3 ~= r3, "math.fmod(1, 0.0) should be NaN, got " .. tostring(r3))

-- Test 4: math.fmod(int, 0) should still error
local ok, err = pcall(math.fmod, 1, 0)
assert(not ok, "math.fmod(1, 0) should error")
assert(err:find("%%0") or err:find("zero"), "error should mention mod by zero")

print("PASS")
