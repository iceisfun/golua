-- test_fuzz_math_rad_huge:
-- math.rad should compute x * (pi / 180), avoiding avoidable overflow.

local got = string.format("%.17g", math.rad(1e308))
assert(got == "1.7453292519943295e+306",
  "math.rad(1e308) mismatch: " .. got)

print("ok")
