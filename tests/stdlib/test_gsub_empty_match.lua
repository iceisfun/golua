-- Test: pm.lua - Empty match semantics (5.3.3+)
-- From: pm.lua
-- What: Tests the 5.3.3+ semantics for empty matches in gsub and gmatch, where empty matches advance correctly through the string.

do   -- new (5.3.3) semantics for empty matches
  assert(string.gsub("a b cd", " *", "-") == "-a-b-c-d-")

  local res = ""
  local sub = "a  \nbc\t\td"
  local i = 1
  for p, e in string.gmatch(sub, "()%s*()") do
    res = res .. string.sub(sub, i, p - 1) .. "-"
    i = e
  end
  assert(res == "-a-b-c-d-")
end
