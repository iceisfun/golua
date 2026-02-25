-- test_math_type2: math.type edge cases

-- non-number types return nil per Lua 5.4
assert(math.type("123") == nil, "math.type('123') should be nil")
assert(math.type(nil) == nil, "math.type(nil) should be nil")

-- number types
assert(math.type(123) == "integer", "math.type(123)")
assert(math.type(123.0) == "float", "math.type(123.0)")
