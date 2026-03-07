-- Float modulo by zero should produce -nan (matching C's fmod)
assert(tostring(1.0 % 0) == "-nan", "1.0 %% 0 should be -nan, got: " .. tostring(1.0 % 0))
assert(tostring(0.0 % 0) == "-nan", "0.0 %% 0 should be -nan, got: " .. tostring(0.0 % 0))
assert(tostring(-1.0 % 0) == "-nan", "-1.0 %% 0 should be -nan, got: " .. tostring(-1.0 % 0))
-- Regular NaN from 0/0 should also be -nan
assert(tostring(0/0) == "-nan", "0/0 should be -nan")
