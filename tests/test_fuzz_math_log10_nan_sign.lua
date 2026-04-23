-- math.log(x, 10) with negative x should produce positive NaN (matching lua5.5.0 / glibc log10).
-- Go's math.Log10 returns a signed NaN; we canonicalize it.
local r = math.log(-1, 10)
assert(tostring(r) == "nan", "math.log(-1, 10) should be nan, got: " .. tostring(r))
-- Sign bit check via string.pack: positive NaN has high byte 0x7F, negative NaN has 0xFF.
local b = string.pack(">d", r):byte(1)
assert(b == 127, "math.log(-1, 10) high byte should be 127 (positive NaN), got: " .. tostring(b))
