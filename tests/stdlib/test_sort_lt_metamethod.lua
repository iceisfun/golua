-- Test: table.sort with __lt metamethod
-- From: sort.lua
-- What: Tests that table.sort correctly uses the __lt metamethod when sorting tables with a custom less-than comparison.

do
local function check (a, f)
  f = f or function (x,y) return x<y end;
  for n = #a, 2, -1 do
    assert(not f(a[n], a[n-1]))
  end
end

local tt = {__lt = function (a,b) return a.val < b.val end}
local a = {}
for i=1,10 do  a[i] = {val=math.random(100)}; setmetatable(a[i], tt); end
table.sort(a)
check(a, tt.__lt)
check(a)
end
