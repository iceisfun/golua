-- test_fuzz_math_deg_huge:
-- math.deg should compute x * (180 / pi), avoiding avoidable overflow:
-- the naive x*180/pi overflows to +Inf for x where x*(180/pi) is still finite.

local got = string.format("%.17g", math.deg(2e306))
assert(got == "1.1459155902616465e+308",
  "math.deg(2e306) mismatch: " .. got)

print("ok")
