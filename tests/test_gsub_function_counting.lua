-- Test: pm.lua - string.gsub with function replacement and counting
-- From: pm.lua
-- What: Tests string.gsub with function replacements using sequential table values and count limiting (4th argument).

do
  local t = {"apple", "orange", "lime"; n=0}
  assert(string.gsub("x and x and x", "x", function () t.n=t.n+1; return t[t.n] end)
          == "apple and orange and lime")

  t = {n=0}
  string.gsub("first second word", "%w%w*", function (w) t.n=t.n+1; t[t.n] = w end)
  assert(t[1] == "first" and t[2] == "second" and t[3] == "word" and t.n == 3)

  t = {n=0}
  assert(string.gsub("first second word", "%w+",
           function (w) t.n=t.n+1; t[t.n] = w end, 2) == "first second word")
  assert(t[1] == "first" and t[2] == "second" and t[3] == undef)
end
