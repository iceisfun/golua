-- Test: utf8.char basic and limits
-- From: utf8.lua
-- What: Tests utf8.char() with no arguments, basic characters, max codepoint, and error handling for out-of-range values.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

assert(utf8.char() == "")
assert(utf8.char(0, 97, 98, 99, 1) == "\0abc\1")

assert(utf8.codepoint(utf8.char(0x10FFFF)) == 0x10FFFF)
assert(utf8.codepoint(utf8.char(0x7FFFFFFF), 1, 1, true) == (1<<31) - 1)

checkerror("value out of range", utf8.char, 0x7FFFFFFF + 1)
checkerror("value out of range", utf8.char, -1)
end
