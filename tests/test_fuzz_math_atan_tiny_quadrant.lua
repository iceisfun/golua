-- test_fuzz_math_atan_tiny_quadrant:
-- math.atan(y, x) must preserve the sign of a tiny negative y in quadrant III.

local got = string.format("%.17g", math.atan(-1e-308, -1e308))
assert(got == "-3.1415926535897931",
  "math.atan(-1e-308, -1e308) mismatch: " .. got)

print("ok")
