-- Test: pm.lua - Balanced match pattern (%b)
-- From: pm.lua
-- What: Tests the `%b` balanced match pattern for parentheses and quoted strings.

do
  local function isbalanced (s)
    return not string.find(string.gsub(s, "%b()", ""), "[()]")
  end

  assert(isbalanced("(9 ((8))(\0) 7) \0\0 a b ()(c)() a"))
  assert(not isbalanced("(9 ((8) 7) a b (\0 c) a"))
  assert(string.gsub("alo 'oi' alo", "%b''", '"') == 'alo " alo')
end
