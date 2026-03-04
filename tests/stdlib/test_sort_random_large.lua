-- Test: table.sort with random data and timing
-- From: sort.lua
-- What: Tests table.sort performance and correctness on large random arrays (50K elements), re-sorting already-sorted arrays, and inverse sorting with a custom comparator.

do
local function check (a, f)
  f = f or function (x,y) return x<y end;
  for n = #a, 2, -1 do
    assert(not f(a[n], a[n-1]))
  end
end

local limit = 50000

local a = {}
for i=1,limit do
  a[i] = math.random()
end

table.sort(a)
check(a)

table.sort(a)
check(a)

a = {}
for i=1,limit do
  a[i] = math.random()
end

table.sort(a, function(x,y) return y<x end)
check(a, function(x,y) return y<x end)
end
