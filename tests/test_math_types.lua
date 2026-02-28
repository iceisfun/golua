-- test_math_types: math.type and math.tointeger

-- math.type
do
    assert(math.type(123) == "integer", "math.type(123)")
    assert(math.type(123.0) == "float", "math.type(123.0)")

    -- non-number types return nil per Lua 5.4
    assert(math.type("123") == nil, "math.type('123') should be nil")
    assert(math.type("x") == nil, "math.type on string should be nil")
    assert(math.type(nil) == nil, "math.type(nil) should be nil")
    assert(math.type(true) == nil, "math.type on boolean should be nil")
    assert(math.type({}) == nil, "math.type on table should be nil")
    assert(math.type("x") ~= false, "math.type should return nil, not false")
end

-- math.tointeger
do
    -- float with integer value
    assert(math.tointeger(56.0) == 56, "tointeger(56.0)")
    assert(math.tointeger(5.0) == 5, "numeric input should still convert")

    -- float with fractional part
    assert(math.tointeger(1.5) == nil, "tointeger(1.5) should be nil")

    -- string input: Lua 5.4 coerces strings via lua_tointegerx
    assert(math.tointeger("-123") == -123, "tointeger('-123') should be -123")
    assert(math.tointeger("5") == 5, "math.tointeger('5') should be 5")
    assert(math.tointeger("  42  ") == 42, "math.tointeger('  42  ') should be 42")
    assert(math.tointeger("0x10") == 16, "math.tointeger('0x10') should be 16")
    assert(math.tointeger("0xA") == 10, "tointeger should coerce hex string")

    -- Float strings that are exact integers should coerce
    assert(math.tointeger("5.0") == 5, "math.tointeger('5.0') should be 5")
    assert(math.tointeger("1e3") == 1000, "math.tointeger('1e3') should be 1000")

    -- Non-integer values should return nil
    assert(math.tointeger("3.14") == nil, "math.tointeger('3.14') should be nil")
    assert(math.tointeger("5.5") == nil, "non-integer float string should return nil")
    assert(math.tointeger("hello") == nil, "math.tointeger('hello') should be nil")

    -- Non-number types should return nil
    assert(math.tointeger({}) == nil, "tointeger({}) should be nil")
    assert(math.tointeger(true) == nil, "tointeger(true) should be nil")
    assert(math.tointeger(nil) == nil, "tointeger(nil) should be nil")
end
