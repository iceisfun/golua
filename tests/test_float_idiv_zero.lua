-- BROKEN: Float floor division by zero should return inf, not error
-- Lua 5.4: integer // 0 errors, but float // 0.0 returns inf/-inf/nan.
-- Currently both cases error with "attempt to perform 'n//0'".

assert(1.0 // 0.0 == math.huge, "1.0 // 0.0 should be inf")
assert(-1.0 // 0.0 == -math.huge, "-1.0 // 0.0 should be -inf")
assert(0.0 // 0.0 ~= 0.0 // 0.0, "0.0 // 0.0 should be NaN (not equal to itself)")
