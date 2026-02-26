-- Bug: math.modf(NaN) returns an integer (MinInt64) for the integer part.
-- Lua 5.4 returns (NaN, NaN) — both parts should be NaN floats.

local intPart, fracPart = math.modf(0/0)
assert(intPart ~= intPart,
  "math.modf(NaN) integer part should be NaN, got: " .. tostring(intPart))
assert(fracPart ~= fracPart,
  "math.modf(NaN) fractional part should be NaN, got: " .. tostring(fracPart))

-- Verify infinity still works correctly too
local intInf, fracInf = math.modf(math.huge)
assert(intInf == math.huge,
  "math.modf(inf) integer part should be inf, got: " .. tostring(intInf))
assert(fracInf == 0.0,
  "math.modf(inf) fractional part should be 0.0, got: " .. tostring(fracInf))

local intNinf, fracNinf = math.modf(-math.huge)
assert(intNinf == -math.huge,
  "math.modf(-inf) integer part should be -inf, got: " .. tostring(intNinf))
assert(fracNinf == -0.0 or fracNinf == 0.0,
  "math.modf(-inf) fractional part should be -0.0, got: " .. tostring(fracNinf))

print("PASS")
