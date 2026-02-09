-- test_math_type2: math.type edge cases

-- non-number types return false (not nil) per Lua 5.4
assert(math.type("123") == false, "math.type('123') should be false")
assert(math.type(nil) == false, "math.type(nil) should be false")

-- number types
assert(math.type(123) == "integer", "math.type(123)")
assert(math.type(123.0) == "float", "math.type(123.0)")
