-- Bug #7: math.tointeger doesn't coerce strings
-- Lua 5.4's lua_tointegerx handles string-to-number conversion.

assert(math.tointeger("5") == 5, "tointeger should coerce integer string")
assert(math.tointeger("5.0") == 5, "tointeger should coerce float string with integer value")
assert(math.tointeger("0xA") == 10, "tointeger should coerce hex string")

-- Non-integer values should return nil
assert(math.tointeger("5.5") == nil, "non-integer float string should return nil")
assert(math.tointeger("hello") == nil, "non-numeric string should return nil")

-- Non-string, non-number types should return nil
assert(math.tointeger(true) == nil)
assert(math.tointeger(nil) == nil)
