-- Test: constant folding of modulo with negative zero
-- The constant folder must preserve -0.0 sign in modulo results.

-- These are constant-folded at compile time:
assert(1/(-0.0 % 1) == -math.huge, "-0.0 % 1 should be -0.0")
assert(1/(-0.0 % -1) == -math.huge, "-0.0 % -1 should be -0.0")
assert(1/(-0.0 % 2) == -math.huge, "-0.0 % 2 should be -0.0")

-- Verify via tostring
assert(tostring(-0.0 % 1) == "-0.0", "tostring(-0.0 % 1) should be '-0.0'")
assert(tostring(-0.0 % -1) == "-0.0", "tostring(-0.0 % -1) should be '-0.0'")

print("OK")
