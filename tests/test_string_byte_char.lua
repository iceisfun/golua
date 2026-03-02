-- Test: string.byte and string.char
-- From: strings.lua
-- What: Tests string.byte/string.char for ASCII, high-byte values, null bytes, multi-return, negative indices, out-of-range indices, and round-tripping.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

assert(string.byte("a") == 97)
assert(string.byte("\xe4") > 127)
assert(string.byte(string.char(255)) == 255)
assert(string.byte(string.char(0)) == 0)
assert(string.byte("\0") == 0)
assert(string.byte("\0\0alo\0x", -1) == string.byte('x'))
assert(string.byte("ba", 2) == 97)
assert(string.byte("\n\n", 2, -1) == 10)
assert(string.byte("\n\n", 2, 2) == 10)
assert(string.byte("") == nil)
assert(string.byte("hi", -3) == nil)
assert(string.byte("hi", 3) == nil)
assert(string.byte("hi", 9, 10) == nil)
assert(string.byte("hi", 2, 1) == nil)
assert(string.char() == "")
assert(string.char(0, 255, 0) == "\0\255\0")
assert(string.char(0, string.byte("\xe4"), 0) == "\0\xe4\0")
assert(string.char(string.byte("\xe4l\0\xE3u", 1, -1)) == "\xe4l\0\xE3u")
assert(string.char(string.byte("\xe4l\0\xE3u", 1, 0)) == "")
assert(string.char(string.byte("\xe4l\0\xE3u", -10, 100)) == "\xe4l\0\xE3u")

checkerror("out of range", string.char, 256)
checkerror("out of range", string.char, -1)
checkerror("out of range", string.char, math.maxinteger)
checkerror("out of range", string.char, math.mininteger)
end
