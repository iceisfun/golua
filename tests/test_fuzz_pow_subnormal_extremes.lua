-- broken_fuzz_pow_subnormal_extremes:
-- vm/vm_pow.go PowWithSubnormalFix produces wrong magnitudes on extreme y.
--
-- BROKEN: The 2026-04-23 PowWithSubnormalFix correctly handles deep-subnormal
-- bases like 5e-324, but breaks on subnormal bases like 1e-308 (just below
-- DBL_MIN ≈ 2.2250738585072014e-308) when y has very large magnitude. The
-- fix decomposes x = m * 2^-1074 and computes m^y * 2^(-1074*y); the second
-- factor overflows or underflows for large |y|, and Inf*0 collapses to NaN.
--
-- Failing cases at base = 1e-308:
--   y = -1                -> golua: inf       ref: 1e+308
--   y = -1.0              -> golua: inf       ref: 1e+308
--   y = math.huge         -> golua: -nan      ref: 0.0
--   y = -math.huge        -> golua: -nan      ref: inf
--   y = math.maxinteger   -> golua: -nan      ref: 0.0
--   y = math.mininteger   -> golua: -nan      ref: inf
--   y = 100 (large but finite) -> golua: -nan ref: 0.0
--
-- Mid-range y values and the deep-subnormal cases (5e-324) still work — the
-- bug is at the |y| ≫ 1 edges of the 2^(-1074*y) factor.
--
-- Reference (lua5.5.0 and lua 5.4.8):
--   1e-308 ^ -1 == 1e+308
--   1e-308 ^ math.huge == 0.0
--   1e-308 ^ -math.huge == math.huge
--   1e-308 ^ 100 == 0.0
--
-- golua today: see assertions below — wrong magnitudes / NaNs.
--
-- Discovered: differential fuzz 2026-05-04 (math wave-1 agent). Same code
-- path as the round-1 PowWithSubnormalFix; this exposes additional edges
-- the original fix didn't anticipate.

local x = 1e-308

-- The big one: y = -1 should give the exact reciprocal, ~1e+308.
local r1 = x ^ -1
assert(math.abs(r1 - 1e+308) / 1e+308 < 0.01,
  string.format("1e-308 ^ -1: expected ≈1e+308, got %g", r1))

-- y = -1.0 (float) — same expected
local r2 = x ^ -1.0
assert(math.abs(r2 - 1e+308) / 1e+308 < 0.01,
  string.format("1e-308 ^ -1.0: expected ≈1e+308, got %g", r2))

-- y = math.huge: 0 < x < 1, so x^+inf = 0
local r3 = x ^ math.huge
assert(r3 == 0.0,
  string.format("1e-308 ^ inf: expected 0.0, got %g", r3))

-- y = -math.huge: 0 < x < 1, so x^-inf = inf
local r4 = x ^ -math.huge
assert(r4 == math.huge,
  string.format("1e-308 ^ -inf: expected inf, got %g", r4))

-- y = math.maxinteger (very large positive int): 0 < x < 1, underflow to 0
local r5 = x ^ math.maxinteger
assert(r5 == 0.0,
  string.format("1e-308 ^ MAXINT: expected 0.0, got %g", r5))

-- y = math.mininteger (very large negative int): inverse underflow, → inf
local r6 = x ^ math.mininteger
assert(r6 == math.huge,
  string.format("1e-308 ^ MININT: expected inf, got %g", r6))

-- y = 100 — well-defined: 0 < x < 1, raised to large positive int → 0
local r7 = x ^ 100
assert(r7 == 0.0,
  string.format("1e-308 ^ 100: expected 0.0, got %g", r7))

print("ok")
