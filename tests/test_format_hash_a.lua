-- Test: string.format "%#a" forces decimal point with # flag
-- The alternate form flag (#) with %a/%A should force a decimal point
-- even when there are no fractional hex digits.

-- Exact powers of 2 have no fractional digits by default
assert(string.format("%a", 1.0) == "0x1p+0", "baseline: no decimal point without #")
assert(string.format("%#a", 1.0) == "0x1.p+0", "# flag should force decimal point for 1.0")
assert(string.format("%#a", 2.0) == "0x1.p+1", "# flag should force decimal point for 2.0")
assert(string.format("%#a", 0.5) == "0x1.p-1", "# flag should force decimal point for 0.5")

-- %#A (uppercase) should also work
assert(string.format("%#A", 1.0) == "0X1.P+0", "# flag with %A")

-- When there are already fractional digits, # flag has no additional effect
assert(string.format("%#a", 1.5) == "0x1.8p+0", "# flag with existing fractional digits")

-- With explicit precision, # should still work
assert(string.format("%#.0a", 1.0) == "0x1.p+0", "# flag with .0 precision")

print("OK")
