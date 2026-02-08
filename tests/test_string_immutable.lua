-- Test string immutability
local s = "abc"
local t = s
t = t .. "d"

assert(s == "abc")
assert(t == "abcd")
