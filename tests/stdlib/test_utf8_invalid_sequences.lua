-- Test: Invalid UTF-8 sequences
-- From: utf8.lua
-- What: Tests that various invalid UTF-8 byte sequences are properly rejected.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

local function invalid (s)
  checkerror("invalid UTF%-8 code", utf8.codepoint, s)
  assert(not utf8.len(s))
end

invalid("\xF4\x9F\xBF\xBF")
invalid("\u{D800}")
invalid("\u{DFFF}")
invalid("\xC0\x80")
invalid("\xC1\xBF")
invalid("\xE0\x9F\xBF")
invalid("\xF0\x8F\xBF\xBF")
invalid("\x80")
invalid("\xBF")
invalid("\xFE")
invalid("\xFF")
end
