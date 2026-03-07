-- tonumber() should reject "infinity", "Infinity", "INFINITY" and return nil.
-- Lua 5.4 only accepts numeric strings that C's strtod/strtol would parse as
-- finite numbers. Go's strconv.ParseFloat accepts "Infinity" natively, so
-- GoLua must explicitly filter it out.
--
-- Also verifies that implicit string-to-number coercion in arithmetic rejects
-- "inf", "infinity", "nan", and "NaN" — these should all produce errors, not
-- silently coerce to special float values.

-- tonumber explicit calls
assert(tonumber("infinity") == nil, "tonumber('infinity') should be nil")
assert(tonumber("Infinity") == nil, "tonumber('Infinity') should be nil")
assert(tonumber("-infinity") == nil, "tonumber('-infinity') should be nil")
assert(tonumber("+Infinity") == nil, "tonumber('+Infinity') should be nil")
assert(tonumber("INFINITY") == nil, "tonumber('INFINITY') should be nil")
assert(tonumber("+INFINITY") == nil, "tonumber('+INFINITY') should be nil")
assert(tonumber("-INFINITY") == nil, "tonumber('-INFINITY') should be nil")

-- "inf" / "nan" should also be rejected
assert(tonumber("inf") == nil, "tonumber('inf') should be nil")
assert(tonumber("Inf") == nil, "tonumber('Inf') should be nil")
assert(tonumber("INF") == nil, "tonumber('INF') should be nil")
assert(tonumber("nan") == nil, "tonumber('nan') should be nil")
assert(tonumber("NaN") == nil, "tonumber('NaN') should be nil")
assert(tonumber("NAN") == nil, "tonumber('NAN') should be nil")

-- With leading/trailing whitespace
assert(tonumber(" infinity ") == nil, "tonumber(' infinity ') should be nil")
assert(tonumber(" inf ") == nil, "tonumber(' inf ') should be nil")
assert(tonumber(" nan ") == nil, "tonumber(' nan ') should be nil")

-- Arithmetic coercion should also reject these
local cases = {"inf", "Inf", "INF", "infinity", "Infinity", "nan", "NaN"}
for _, s in ipairs(cases) do
  local ok, result = pcall(function() return s + 0 end)
  assert(not ok, "'" .. s .. "' + 0 should error, but got: " .. tostring(result))
end

-- Verify that normal numeric strings still work
assert(tonumber("42") == 42)
assert(tonumber("3.14") == 3.14)
assert(tonumber("0xff") == 255)
assert(tonumber("1e10") == 1e10)

