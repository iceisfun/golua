-- Test special floating point values
assert(0/0 ~= 0/0)      -- NaN
assert(1/0 > 0)         -- +inf
assert(-1/0 < 0)        -- -inf
