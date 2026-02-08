-- Test string.match with captures
local a, b = string.match("hello123", "(%a+)(%d+)")
assert(a == "hello")
assert(b == "123")
