-- Test: Error in initial position for utf8.offset
-- From: utf8.lua
-- What: Tests that utf8.offset raises errors for invalid starting positions.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

checkerror("position out of bounds", utf8.offset, "abc", 1, 5)
checkerror("position out of bounds", utf8.offset, "abc", 1, -4)
checkerror("position out of bounds", utf8.offset, "", 1, 2)
checkerror("position out of bounds", utf8.offset, "", 1, -1)
checkerror("continuation byte", utf8.offset, "𦧺", 1, 2)
checkerror("continuation byte", utf8.offset, "𦧺", 1, 2)
checkerror("continuation byte", utf8.offset, "\x80", 1)
end
