-- test_math_tointeger2: math.tointeger edge cases

-- float with integer value
assert(math.tointeger(56.0) == 56, "tointeger(56.0)")

-- string input: Lua 5.4 coerces strings via lua_tointegerx
assert(math.tointeger("-123") == -123, "tointeger('-123') should be -123")

-- table input
assert(math.tointeger({}) == nil, "tointeger({}) should be nil")

-- boolean input
assert(math.tointeger(true) == nil, "tointeger(true) should be nil")

-- float with fractional part
assert(math.tointeger(1.5) == nil, "tointeger(1.5) should be nil")
