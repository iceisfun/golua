-- Test: attrib.lua - Multiple return value unpacking in assignments
-- From: attrib.lua
-- What: Tests how multiple return values interact with multiple assignment

do
  local f = function (n) local x = {}; for i=1,n do x[i]=i end;
                         return table.unpack(x) end;
  local a,b,c
  a,b = 0, f(1)
  assert(a == 0 and b == 1)
  a,b,c = 0,5,f(4)
  assert(a==0 and b==5 and c==1)
  a,b,c = 0,5,f(0)
  assert(a==0 and b==5 and c==nil)
end
