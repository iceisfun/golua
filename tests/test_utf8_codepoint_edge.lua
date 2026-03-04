-- Test: utf8.codepoint edge cases and surrogates
-- From: utf8.lua
-- What: Tests utf8.codepoint with partial reads, errors, empty ranges, and surrogate codepoints in strict/non-strict modes.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

do
  local s = "áéí\128"
  local t = {utf8.codepoint(s,1,#s - 1)}
  assert(#t == 3 and t[1] == 225 and t[2] == 233 and t[3] == 237)
  checkerror("invalid UTF%-8 code", utf8.codepoint, s, 1, #s)
  checkerror("out of bounds", utf8.codepoint, s, #s + 1)
  t = {utf8.codepoint(s, 4, 3)}
  assert(#t == 0)
  checkerror("out of bounds", utf8.codepoint, s, -(#s + 1), 1)
  checkerror("out of bounds", utf8.codepoint, s, 1, #s + 1)
  assert(utf8.codepoint("\u{D7FF}") == 0xD800 - 1)
  assert(utf8.codepoint("\u{E000}") == 0xDFFF + 1)
  assert(utf8.codepoint("\u{D800}", 1, 1, true) == 0xD800)
  assert(utf8.codepoint("\u{DFFF}", 1, 1, true) == 0xDFFF)
  assert(utf8.codepoint("\u{7FFFFFFF}", 1, 1, true) == 0x7FFFFFFF)
end
end
