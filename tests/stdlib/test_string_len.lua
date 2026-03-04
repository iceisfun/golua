-- Test: string.len and # operator
-- From: strings.lua
-- What: Tests string.len and the # length operator on empty strings, strings with null bytes, and regular strings.

do
assert(string.len("") == 0)
assert(string.len("\0\0\0") == 3)
assert(string.len("1234567890") == 10)

assert(#"" == 0)
assert(#"\0\0\0" == 3)
assert(#"1234567890" == 10)
end
