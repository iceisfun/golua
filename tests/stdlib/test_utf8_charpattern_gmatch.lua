-- Test: charpattern iteration with gmatch and offset
-- From: utf8.lua
-- What: Tests that iterating with string.gmatch using utf8.charpattern matches utf8.offset positions.

do
local x = "日本語a-4\0éó"
local i = 0
for p, c in string.gmatch(x, "()(" .. utf8.charpattern .. ")") do
  i = i + 1
  assert(utf8.offset(x, i) == p)
  assert(utf8.len(x, p) == utf8.len(x) - i + 1)
  assert(utf8.len(c) == 1)
  for j = 1, #c - 1 do
    assert(utf8.offset(x, 0, p + j - 1) == p)
  end
end
end
