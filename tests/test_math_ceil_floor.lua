-- test_math_ceil_floor: math.ceil and math.floor edge cases

-- ceil
assert(math.ceil(123) == 123, "ceil(123)")
assert(math.ceil(12.7) == 13, "ceil(12.7)")
assert(math.ceil(-8.2) == -8, "ceil(-8.2)")
assert(math.ceil("888.5673") == 889, "ceil string coercion")

-- floor
assert(math.floor(123) == 123, "floor(123)")
assert(math.floor(12.7) == 12, "floor(12.7)")
assert(math.floor(-8.2) == -9, "floor(-8.2)")
assert(math.floor("888.5673") == 888, "floor string coercion")

-- both should return integer type
assert(math.type(math.floor(3.5)) == "integer", "floor returns integer")
assert(math.type(math.ceil(3.5)) == "integer", "ceil returns integer")

-- no args should error
assert(not pcall(math.ceil), "ceil() no args")
assert(not pcall(math.floor), "floor() no args")

-- table arg should error
assert(not pcall(math.ceil, {}), "ceil({}) should error")
assert(not pcall(math.floor, {}), "floor({}) should error")
