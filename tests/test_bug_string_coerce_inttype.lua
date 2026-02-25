-- Bug #3: String-to-number coercion always yields float even for integer strings
-- Lua 5.4 preserves integer type when the string represents an exact integer.

assert(type("5" + 3) == "number" and "5" + 3 == 8)
assert(math.type("5" + 3) == "integer", "integer string + integer should be integer")
assert(math.type("5" * 2) == "integer", "integer string * integer should be integer")
assert(math.type(10 - "3") == "integer", "integer - integer string should be integer")
assert(math.type("0xA" + 1) == "integer", "hex string + integer should be integer")

-- Float strings should remain float
assert(math.type("5.0" + 3) == "float", "float string + integer should be float")
assert(math.type("1e2" + 0) == "float", "scientific notation string should be float")
