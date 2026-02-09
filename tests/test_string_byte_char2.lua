-- test_string_byte_char2: string.byte and string.char edge cases

-- byte basics
do
    local s = "hello"
    assert(string.byte(s) == 104, "byte default first")
    assert(string.byte(s, -1) == 111, "byte negative index")

    local a, b, c = string.byte(s, 2, 4)
    assert(a == 101 and b == 108 and c == 108, "byte range")

    local d, e = string.byte(s, -5, 2)
    assert(d == 104 and e == 101, "byte neg start to pos end")

    local f, g = string.byte(s, -2, -1)
    assert(f == 108 and g == 111, "byte neg range")
end

-- byte as method
do
    assert(("hello"):byte(3) == 108, "byte method syntax")
end

-- byte error cases
assert(not pcall(string.byte), "byte no args")
assert(not pcall(string.byte, {}), "byte non-string")
assert(not pcall(string.byte, "xxx", true), "byte non-int index")

-- char basics
assert(string.char(65, 66, 67) == "ABC", "char basic")

-- char error cases
assert(not pcall(string.char, -1), "char negative")
assert(not pcall(string.char, 256), "char overflow")
assert(not pcall(string.char, 1, 2, "x"), "char non-int")
