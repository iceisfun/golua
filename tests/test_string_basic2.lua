-- test_string_basic2: len, lower, upper, rep, reverse, sub

-- len
assert(string.len("abc") == 3 and string.len("") == 0, "len basic")
assert(not pcall(string.len), "len no args")

-- lower / upper
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

-- rep error cases
assert(not pcall(string.rep), "rep no args")

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
