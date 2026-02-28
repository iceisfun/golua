-- test_string_basics: len, lower, upper, rep, reverse, sub, byte, char, immutability

-- len/lower/upper/rep/reverse/sub
assert(string.len("abc") == 3 and string.len("") == 0, "len basic")
assert(not pcall(string.len), "len no args")
assert(string.lower("ABCdef123") == "abcdef123", "lower")
assert(string.upper("ABCdef123") == "ABCDEF123", "upper")
assert(not pcall(string.lower), "lower no args")
assert(not pcall(string.upper), "upper no args")

-- rep
assert(string.rep("xy", 0) == "", "rep 0")
assert(string.rep("xy", 1) == "xy", "rep 1")
assert(string.rep("xy", 2) == "xyxy", "rep 2")
assert(string.rep("xy", 3) == "xyxyxy", "rep 3")

-- rep with separator
assert(string.rep("xy", 0, "--") == "", "rep sep 0")
assert(string.rep("xy", 1, "--") == "xy", "rep sep 1")
assert(string.rep("xy", 2, "--") == "xy--xy", "rep sep 2")
assert(string.rep("xy", 3, "--") == "xy--xy--xy", "rep sep 3")

-- rep edge cases
assert(not pcall(string.rep), "rep no args")
assert(string.rep("xx", -1) == "", "rep with negative count should return empty")
assert(string.rep("abc", 0) == "", "rep with zero count should return empty")
assert(string.rep("x", 3) == "xxx", "rep basic")

-- reverse
assert(string.reverse("EGASSEM TERCES") == "SECRET MESSAGE", "reverse")
assert(string.reverse("12345") == "54321", "reverse digits")
assert(not pcall(string.reverse), "reverse no args")

-- sub
do
    local s = "abc"
    assert(s:sub(1) == "abc", "sub from 1")
    assert(s:sub(2) == "bc", "sub from 2")
    assert(s:sub(3) == "c", "sub from 3")
    assert(s:sub(4) == "", "sub past end")
    assert(s:sub(-2) == "bc", "sub neg")
    assert(s:sub(2, 3) == "bc", "sub range")
    assert(s:sub(3, 6) == "c", "sub past end range")
end
assert(not pcall(string.sub), "sub no args")

-- byte/char
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
do
    assert(("hello"):byte(3) == 108, "byte method syntax")
end
assert(not pcall(string.byte), "byte no args")
assert(not pcall(string.byte, {}), "byte non-string")
assert(not pcall(string.byte, "xxx", true), "byte non-int index")
assert(string.char(65, 66, 67) == "ABC", "char basic")
assert(not pcall(string.char, -1), "char negative")
assert(not pcall(string.char, 256), "char overflow")
assert(not pcall(string.char, 1, 2, "x"), "char non-int")

-- Immutability
do
    local s = "abc"
    local t = s
    t = t .. "d"
    assert(s == "abc")
    assert(t == "abcd")
end
