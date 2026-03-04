-- Test: pm.lua - Captures with string.match
-- From: pm.lua
-- What: Tests capture groups in string.match including nested captures, position captures, and UTF-8 character captures.

do
  assert(string.match("alo xyzK", "(%w+)K") == "xyz")
  assert(string.match("254 K", "(%d*)K") == "")
  assert(string.match("alo ", "(%w*)$") == "")
  assert(not string.match("alo ", "(%w+)$"))
  assert(string.find("(\195\161lo)", "%(\195\161") == 1)
  local a, b, c, d  = string.match('0123456789', '(.+(.?)())')
  assert(a == '0123456789' and b == '' and c == 11 and d == nil)
end
