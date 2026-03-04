-- Test: Basic string assignment with whitespace and embedded nulls
-- From: literals.lua
-- What: Tests that the scanner handles various whitespace characters around assignments and that strings can contain embedded null bytes.

do
  local function dostring (x) return assert(load(x), "")() end

  dostring("x \v\f = \t\r 'a\0a' \v\f\f")
  assert(x == 'a\0a' and string.len(x) == 3)
  _G.x = nil
end
