-- Test: pm.lua - Big string pattern matching
-- From: pm.lua
-- What: Tests pattern matching on very large strings (300K+ characters) including greedy, optional, and lazy quantifiers, and a gsub that should fail on missing replacement.

do
  local a = string.rep('a', 300000)
  assert(string.find(a, '^a*.?$'))
  assert(not string.find(a, '^a*.?b$'))
  assert(string.find(a, '^a-.?$'))

  -- bug in 5.1.2
  a = string.rep('a', 10000) .. string.rep('b', 10000)
  assert(not pcall(string.gsub, a, 'b'))
end
