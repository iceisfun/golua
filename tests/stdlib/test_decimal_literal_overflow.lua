-- Test: decimal integer literals that overflow int64 promote to float
-- Lua 5.4 converts overflowing decimal literals to float, not wrapping integers.

-- 2^63 (math.maxinteger + 1) overflows int64 → float
assert(math.type(9223372036854775808) == "float", "2^63 should be float")
assert(9223372036854775808 > 0, "2^63 should be positive")

-- 2^64 - 1 also overflows int64 → float
assert(math.type(18446744073709551615) == "float", "2^64-1 should be float")
assert(18446744073709551615 > 0, "2^64-1 should be positive")

-- 2^63 - 1 = math.maxinteger should still be integer
assert(math.type(9223372036854775807) == "integer", "maxinteger should be integer")

-- -2^63 literal: the literal 9223372036854775808 is already float,
-- so unary minus produces a float
assert(math.type(-9223372036854775808) == "float", "-2^63 literal should be float")

-- Very large decimal should be float (not error)
assert(math.type(99999999999999999999) == "float", "large decimal should be float")

