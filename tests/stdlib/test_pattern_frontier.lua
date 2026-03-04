-- Test: pm.lua - Frontier pattern (%f)
-- From: pm.lua
-- What: Tests the frontier pattern `%f[set]` which matches an empty string at a position where the next character is in the set but the previous is not.

do
  assert(string.gsub("aaa aa a aaa a", "%f[%w]a", "x") == "xaa xa x xaa x")
  assert(string.gsub("[[]] [][] [[[[", "%f[[].", "x") == "x[]] x]x] x[[[")
  assert(string.gsub("01abc45de3", "%f[%d]", ".") == ".01abc.45de.3")
  assert(string.gsub("01abc45 de3x", "%f[%D]%w", ".") == "01.bc45 de3.")
  assert(string.gsub("function", "%f[\1-\255]%w", ".") == ".unction")
  assert(string.gsub("function", "%f[^\1-\255]", ".") == "function.")

  assert(string.find("a", "%f[a]") == 1)
  assert(string.find("a", "%f[^%z]") == 1)
  assert(string.find("a", "%f[^%l]") == 2)
  assert(string.find("aba", "%f[a%z]") == 3)
  assert(string.find("aba", "%f[%z]") == 4)
  assert(not string.find("aba", "%f[%l%z]"))
  assert(not string.find("aba", "%f[^%l%z]"))

  local i, e = string.find(" alo aalo allo", "%f[%S].-%f[%s].-%f[%S]")
  assert(i == 2 and e == 5)
  local k = string.match(" alo aalo allo", "%f[%S](.-%f[%s].-%f[%S])")
  assert(k == 'alo ')

  local a = {1, 5, 9, 14, 17,}
  for k in string.gmatch("alo alo th02 is 1hat", "()%f[%w%d]") do
    assert(table.remove(a, 1) == k)
  end
  assert(#a == 0)
end
