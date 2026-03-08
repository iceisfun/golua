-- string.format %a and %q should format subnormal floats in the IEEE 754
-- denormalized form: exponent fixed at -1022 with leading zeros in mantissa.
-- GoLua incorrectly normalizes subnormals (e.g., 0x1p-1074 instead of
-- 0x0.0000000000001p-1022).

-- Smallest subnormal
local s1 = string.format("%a", 5e-324)
assert(s1 == "0x0.0000000000001p-1022",
  "format %%a of 5e-324: expected 0x0.0000000000001p-1022, got " .. s1)

-- Largest subnormal (just below smallest normal)
local s2 = string.format("%a", 2.2250738585072009e-308)
assert(s2 == "0x0.fffffffffffffp-1022",
  "format %%a of largest subnormal: expected 0x0.fffffffffffffp-1022, got " .. s2)

-- %q for subnormal should use the denormalized form
local q1 = string.format("%q", 5e-324)
assert(q1 == "0x0.0000000000001p-1022",
  "format %%q of 5e-324: expected 0x0.0000000000001p-1022, got " .. q1)

-- Verify round-trip still works regardless of format
local v1 = tonumber(string.format("%q", 5e-324))
assert(v1 == 5e-324, "round-trip failed for 5e-324")

local v2 = tonumber(string.format("%q", 2.2250738585072009e-308))
assert(v2 == 2.2250738585072009e-308, "round-trip failed for largest subnormal")

print("OK")
