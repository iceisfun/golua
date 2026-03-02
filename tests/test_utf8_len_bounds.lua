-- Test: Error in indices for utf8.len
-- From: utf8.lua
-- What: Tests that utf8.len raises "out of bounds" when indices are outside the string.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

checkerror("out of bounds", utf8.len, "abc", 0, 2)
checkerror("out of bounds", utf8.len, "abc", 1, 4)
end
