-- Test: attrib.lua - Assignments and logical operators
-- From: attrib.lua
-- What: Tests multiple assignment, logical operators (and/or/not), table constructors, and various edge cases

do
  local res, res2 = 27
  local a, b = 1, 2+3
  assert(a==1 and b==5)
  a={}
  local function f() return 10, 11, 12 end
  a.x, b, a[1] = 1, 2, f()
  assert(a.x==1 and b==2 and a[1]==10)
  a[f()], b, a[f()+3] = f(), a, 'x'
  assert(a[10] == 10 and b == a and a[13] == 'x')
end
