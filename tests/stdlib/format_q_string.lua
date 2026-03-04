-- Test: string.format %q (quoted string)
-- From: strings.lua
-- What: Tests string.format("%q", ...) round-trips through load for various strings.

do
-- Round-trip: strings with non-ASCII bytes, newlines, and backslashes
local x = '"\xE3lo"\n\\'
assert(load(string.format('return %q', x))() == x)

-- Round-trip: null byte
assert(load(string.format('return %q', "\0"))() == "\0")

-- Round-trip: mixed control characters and digits
x = "\0\1\0023\5\0009"
assert(load(string.format('return %q', x))() == x)

-- Round-trip: all byte values
for i = 0, 255 do
  local s = string.char(i)
  assert(load(string.format('return %q', s))() == s)
end
end
