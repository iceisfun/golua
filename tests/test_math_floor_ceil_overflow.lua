-- Bug: math.floor and math.ceil incorrectly convert non-finite and
-- out-of-range floats to int64, producing -9223372036854775808 (MinInt64).
-- Lua 5.4 returns float results when the value can't fit in an integer.

-- math.floor with infinity should return infinity (as float)
assert(math.floor(math.huge) == math.huge,
  "math.floor(inf) should be inf, got: " .. tostring(math.floor(math.huge)))

assert(math.floor(-math.huge) == -math.huge,
  "math.floor(-inf) should be -inf, got: " .. tostring(math.floor(-math.huge)))

-- math.floor with NaN should return NaN (as float)
local nanFloor = math.floor(0/0)
assert(nanFloor ~= nanFloor,
  "math.floor(NaN) should be NaN, got: " .. tostring(nanFloor))

-- math.floor with large float that exceeds int64 range
assert(math.floor(2^63) == 2^63,
  "math.floor(2^63) should be 2^63, got: " .. tostring(math.floor(2^63)))

assert(math.floor(1e19) == 1e19,
  "math.floor(1e19) should be 1e19, got: " .. tostring(math.floor(1e19)))

-- math.ceil with infinity
assert(math.ceil(math.huge) == math.huge,
  "math.ceil(inf) should be inf, got: " .. tostring(math.ceil(math.huge)))

assert(math.ceil(-math.huge) == -math.huge,
  "math.ceil(-inf) should be -inf, got: " .. tostring(math.ceil(-math.huge)))

-- math.ceil with NaN
local nanCeil = math.ceil(0/0)
assert(nanCeil ~= nanCeil,
  "math.ceil(NaN) should be NaN, got: " .. tostring(nanCeil))

-- math.ceil with large float
assert(math.ceil(1e19) == 1e19,
  "math.ceil(1e19) should be 1e19, got: " .. tostring(math.ceil(1e19)))

assert(math.ceil(-1e19) == -1e19,
  "math.ceil(-1e19) should be -1e19, got: " .. tostring(math.ceil(-1e19)))

-- math.floor/ceil with values that DO fit should still return integers
assert(math.type(math.floor(2.5)) == "integer",
  "math.floor(2.5) should return integer type")
assert(math.floor(2.5) == 2, "math.floor(2.5) should be 2")

assert(math.type(math.ceil(2.5)) == "integer",
  "math.ceil(2.5) should return integer type")
assert(math.ceil(2.5) == 3, "math.ceil(2.5) should be 3")

print("PASS")
