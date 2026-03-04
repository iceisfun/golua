-- Test: string.format %q with line endings
-- From: literals.lua
-- What: Tests that string.format %q correctly handles strings containing \r, \n, \r\n, and \n\r, and that the result can be round-tripped through load.

do
  local s = "a string with \r and \n and \r\n and \n\r"
  local c = string.format("return %q", s)
  assert(assert(load(c))() == s)
end
