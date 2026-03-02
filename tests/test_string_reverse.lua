-- Test: string.reverse
-- From: strings.lua
-- What: Tests string.reverse on empty strings, strings with null and control bytes.

do
assert(string.reverse"" == "")
assert(string.reverse"\0\1\2\3" == "\3\2\1\0")
assert(string.reverse"\0001234" == "4321\0")

for i=0,30 do assert(string.len(string.rep('a', i)) == i) end
end
