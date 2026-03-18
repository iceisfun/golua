-- Regression: math.modf should preserve integer type for integer inputs
-- Previously, math.modf(math.maxinteger) returned a float due to
-- converting to float64 first (which loses precision for large integers)

local i, f = math.modf(math.maxinteger)
assert(i == math.maxinteger, "modf maxinteger value")
assert(math.type(i) == "integer", "modf maxinteger should be integer, got " .. math.type(i))
assert(f == 0.0, "modf maxinteger fraction")
assert(math.type(f) == "float", "modf maxinteger fraction type")

local i2, f2 = math.modf(math.mininteger)
assert(i2 == math.mininteger, "modf mininteger value")
assert(math.type(i2) == "integer", "modf mininteger should be integer")

local i3, f3 = math.modf(42)
assert(i3 == 42, "modf 42 value")
assert(math.type(i3) == "integer", "modf 42 type")

local i4, f4 = math.modf(0)
assert(i4 == 0, "modf 0 value")
assert(math.type(i4) == "integer", "modf 0 type")

-- Float inputs should still work correctly
local i5, f5 = math.modf(42.5)
assert(i5 == 42, "modf 42.5 integral")
assert(math.type(i5) == "integer", "modf 42.5 integral type")
assert(f5 == 0.5, "modf 42.5 fraction")

local i6, f6 = math.modf(42.0)
assert(i6 == 42, "modf 42.0 integral")
assert(math.type(i6) == "integer", "modf 42.0 integral type")

-- Confirm tostring is correct for maxinteger from modf
assert(tostring(math.modf(math.maxinteger)) == "9223372036854775807",
       "modf maxinteger tostring: " .. tostring(math.modf(math.maxinteger)))

print("OK")
