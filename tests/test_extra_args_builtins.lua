-- Test: calls.lua - Extra arguments to built-in functions
-- From: calls.lua
-- What: Tests that extra arguments to functions like rawget, rawset, math.sin, table.sort are ignored

do
  rawget({}, "x", 1)
  rawset({}, "x", 1, 2)
  assert(math.sin(1,2) == math.sin(1))
  table.sort({10,9,8,4,19,23,0,0}, function (a,b) return a<b end, "extra arg")
end
