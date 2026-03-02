-- Test: pm.lua - Greedy, lazy, and optional quantifiers with string.match
-- From: pm.lua
-- What: Tests `.*`, `.+`, and `.?` quantifiers with string.match, verifying greedy and minimal matching behavior.

do
  assert(string.match("aaab", ".*b") == "aaab")
  assert(string.match("aaa", ".*a") == "aaa")
  assert(string.match("b", ".*b") == "b")

  assert(string.match("aaab", ".+b") == "aaab")
  assert(string.match("aaa", ".+a") == "aaa")
  assert(not string.match("b", ".+b"))

  assert(string.match("aaab", ".?b") == "ab")
  assert(string.match("aaa", ".?a") == "aa")
  assert(string.match("b", ".?b") == "b")
end
