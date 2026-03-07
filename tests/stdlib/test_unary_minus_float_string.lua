-- Test that unary minus on float strings produces floats, not integers
-- Lua 5.4: -"3.0" should be float -3.0, not integer -3

-- Float strings produce float results
assert(math.type(-"3.0") == "float", "expected float, got " .. math.type(-"3.0"))
assert(-"3.0" == -3.0)

-- Integer strings produce integer results
assert(math.type(-"3") == "integer", "expected integer, got " .. math.type(-"3"))
assert(-"3" == -3)

-- Scientific notation is float
assert(math.type(-"3e0") == "float", "expected float for sci notation")
assert(-"3e0" == -3.0)

-- Hex integers are integers
assert(math.type(-"0x3") == "integer", "expected integer for hex int")
assert(-"0x3" == -3)

-- Hex floats are floats
assert(math.type(-"0x1.0p0") == "float", "expected float for hex float")

-- Whitespace-padded float string
assert(math.type(-" 3.0 ") == "float", "expected float for padded float string")

-- Negative zero string
assert(math.type(-"0.0") == "float", "expected float for -0.0 string")

