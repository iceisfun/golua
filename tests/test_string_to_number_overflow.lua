-- Bug: Implicit string-to-number coercion in arithmetic fails for overflow
-- values like "1e999". Lua 5.4 coerces these to inf/-inf (matching tonumber).
-- golua incorrectly errors with "attempt to perform arithmetic on a string value".

-- "1e999" should coerce to inf in arithmetic
assert("1e999" + 0 == math.huge,
  "\"1e999\" + 0 should be inf, got error or wrong value")

-- "-1e999" should coerce to -inf
assert("-1e999" + 0 == -math.huge,
  "\"-1e999\" + 0 should be -inf")

-- "1e309" overflows float64, should be inf
assert("1e309" + 0 == math.huge,
  "\"1e309\" + 0 should be inf")

-- Verify tonumber already handles this correctly (it does)
assert(tonumber("1e999") == math.huge, "tonumber(\"1e999\") should be inf")
assert(tonumber("-1e999") == -math.huge, "tonumber(\"-1e999\") should be -inf")

-- String overflow in other arithmetic operations too
assert("1e999" * 1 == math.huge, "\"1e999\" * 1 should be inf")
assert("1e999" - 0 == math.huge, "\"1e999\" - 0 should be inf")

print("PASS")
