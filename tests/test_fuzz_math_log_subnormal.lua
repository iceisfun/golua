-- test_fuzz_math_log_subnormal:
-- math.log must preserve libm/lua5.5.0 behavior for subnormal inputs.

local got = string.format("%.17g", math.log(1e-308))
assert(got == "-709.19620864216608",
  "math.log(1e-308) mismatch: " .. got)

print("ok")
