-- BUG: string.gsub rejects numeric replacement argument
-- In Lua 5.4, string.gsub accepts a number as the replacement
-- argument, coercing it to a string. GoLua errors with
-- "string/function/table expected, got number".

-- Integer replacement
local r1, n1 = string.gsub("hello", "hello", 42)
assert(r1 == "42", "gsub with integer replacement should work, got: " .. tostring(r1))
assert(n1 == 1, "should report 1 replacement")

-- Float replacement
local r2, n2 = string.gsub("x", "x", 3.14)
assert(r2 == "3.14", "gsub with float replacement should work, got: " .. tostring(r2))

-- Zero replacement
local r3, n3 = string.gsub("abc", "b", 0)
assert(r3 == "a0c", "gsub with 0 replacement should work, got: " .. tostring(r3))

-- Negative number replacement
local r4, n4 = string.gsub("x", "x", -1)
assert(r4 == "-1", "gsub with negative replacement should work, got: " .. tostring(r4))

-- Number replacement with capture reference (captures are ignored for number)
local r5, n5 = string.gsub("abc", "(b)", 99)
assert(r5 == "a99c", "gsub number replacement ignores captures, got: " .. tostring(r5))

-- Non-string/number/function/table should still error
local ok6, e6 = pcall(string.gsub, "x", "x", true)
assert(not ok6, "gsub with boolean should error")
