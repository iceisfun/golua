-- Test: string.find
-- From: strings.lua
-- What: Tests string.find with plain substrings, starting positions, negative indices, plain-mode flag, empty strings, and pattern escaping.

do
assert(string.find("123456789", "345") == 3)
local a,b = string.find("123456789", "345")
assert(string.sub("123456789", a, b) == "345")
assert(string.find("1234567890123456789", "345", 3) == 3)
assert(string.find("1234567890123456789", "345", 4) == 13)
assert(not string.find("1234567890123456789", "346", 4))
assert(string.find("1234567890123456789", ".45", -9) == 13)
assert(not string.find("abcdefg", "\0", 5, 1))
assert(string.find("", "") == 1)
assert(string.find("", "", 1) == 1)
assert(not string.find("", "", 2))
assert(not string.find('', 'aaa', 1))
assert(('alo(.)alo'):find('(.)', 1, 1) == 4)
end
