-- broken_fuzz_pow_nan_sign:
-- The sign bit of NaN results from `^` differs from reference Lua.
--
-- BROKEN: Lua 5.4 and 5.5 agree on the sign-bit propagation of NaN through
-- the `^` operator (matching libm pow on Linux). golua's PowWithSubnormalFix
-- and/or its primary `^` path produces NaNs with the opposite sign on
-- specific input shapes. This is observable as `tostring(x)` printing
-- "nan" vs "-nan", and via `string.format("%a", x)`.
--
-- Cosmetic — Lua spec does not require any particular NaN sign — but it's
-- a real parity gap: golua disagrees with BOTH lua 5.4 and lua 5.5,
-- suggesting it's a golua-specific quirk in the pow path.
--
-- Reference (lua5.5.0 and lua 5.4.8 — both same):
--   (0/0)         ^ 1     -> nan
--   (0/0)         ^ 1.0   -> nan
--   0             ^ (0/0) -> -nan
--   (-0.0)        ^ (0/0) -> -nan
--   math.huge     ^ (0/0) -> -nan
--
-- golua today (signs flipped):
--   (0/0)         ^ 1     -> -nan
--   (0/0)         ^ 1.0   -> -nan
--   0             ^ (0/0) -> nan
--   (-0.0)        ^ (0/0) -> nan
--   math.huge     ^ (0/0) -> nan
--
-- Discovered: differential fuzz 2026-05-04 (math wave-1 agent).
-- Lower priority than the pow_subnormal_extremes finding (cosmetic), but
-- captured here so it doesn't get lost.

-- A NaN's sign is observable via tostring under both 5.4 and 5.5.
local function nansign(x)
  return tostring(x):sub(1, 1) == "-" and "-nan" or "nan"
end

local nan = 0 / 0

assert(nansign(nan ^ 1)        == "nan",  "(0/0) ^ 1 nan-sign mismatch")
assert(nansign(nan ^ 1.0)      == "nan",  "(0/0) ^ 1.0 nan-sign mismatch")
assert(nansign(0 ^ nan)        == "-nan", "0 ^ (0/0) nan-sign mismatch")
assert(nansign((-0.0) ^ nan)   == "-nan", "(-0.0) ^ (0/0) nan-sign mismatch")
assert(nansign(math.huge ^ nan) == "-nan", "huge ^ (0/0) nan-sign mismatch")

print("ok")
