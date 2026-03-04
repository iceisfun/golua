-- Test: errors.lua - assert with various arguments
-- From: errors.lua
-- What: Tests assert() with extra args, no message, non-string messages, and no arguments

do
  local t = {}
  local res, msg = pcall(assert, false, "X", t)
  assert(not res and msg == "X")
  res, msg = pcall(function () assert(false) end)
  local line = string.match(msg, "%w+%.lua:(%d+): assertion failed!$")
  res, msg = pcall(assert, false, t)
  assert(not res and msg == t)
  res, msg = pcall(assert)
  assert(not res and string.find(msg, "value expected"))
end
