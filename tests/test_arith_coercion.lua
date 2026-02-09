-- test_arith_coercion: String-to-number coercion in arithmetic operators

-- Addition
assert("1" + "2" == 3, "str + str")
assert(1 + "3" == 4, "int + str")
assert("2.4" + 2 == 4.4, "float-str + int")

-- Subtraction
assert("2" - "2" == 0, "str - str")

-- Multiplication
assert("3" * 5 == 15, "str * int")
assert(0.1 * "10" == 1.0, "float * str")

-- Division
assert("7" / 2 == 3.5, "str / int")
assert("5" / "10" == 0.5, "str / str")

-- Modulo
assert("3" % 2 == 1, "str % int")

-- Exponentiation
assert(2 ^ "3" == 8, "int ^ str")
assert("5" ^ 2 == 25, "str ^ int")
assert("6" ^ "3" == 216, "str ^ str")

-- Integer division
assert("1" // "2" == 0, "str // str")
assert(7 // "2" == 3, "int // str")
assert("2.4" // 2 == 1, "float-str // int")

-- Non-numeric string should error
assert(not pcall(function() return "a" + "1" end), "non-numeric str + should error")

-- Integer division of float strings by zero gives infinity (float coercion path)
assert(("8" // "0") == math.huge, "str // 0 gives inf")
