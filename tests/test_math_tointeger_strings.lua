-- test_math_tointeger_strings.lua
-- math.tointeger coerces strings per Lua 5.4 (lua_tointegerx handles strings).

-- Integer strings should coerce
assert(math.tointeger("5") == 5, "math.tointeger('5') should be 5")
assert(math.tointeger("  42  ") == 42, "math.tointeger('  42  ') should be 42")
assert(math.tointeger("0x10") == 16, "math.tointeger('0x10') should be 16")

-- Float strings that are exact integers should coerce
assert(math.tointeger("5.0") == 5, "math.tointeger('5.0') should be 5")
assert(math.tointeger("1e3") == 1000, "math.tointeger('1e3') should be 1000")

-- Non-integer values should return nil
assert(math.tointeger("3.14") == nil, "math.tointeger('3.14') should be nil")
assert(math.tointeger("hello") == nil, "math.tointeger('hello') should be nil")

-- Behavior should still work for actual numbers
assert(math.tointeger(5.0) == 5, "numeric input should still convert")
