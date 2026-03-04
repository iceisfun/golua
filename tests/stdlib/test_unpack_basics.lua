-- Test: table.unpack basics
-- From: sort.lua
-- What: Tests table.unpack with full arrays, subranges, empty ranges, single-element unpacking, and select integration.

do
local unpack = table.unpack

local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

checkerror("wrong number of arguments", table.insert, {}, 2, 3, 4)

local x,y,z,a,n
a = {}; local lim = 2000
for i=1, lim do a[i]=i end
assert(select(lim, unpack(a)) == lim and select('#', unpack(a)) == lim)
x = unpack(a)
assert(x == 1)
x = {unpack(a)}
assert(#x == lim and x[1] == 1 and x[lim] == lim)
x = {unpack(a, lim-2)}
assert(#x == 3 and x[1] == lim-2 and x[3] == lim)
x = {unpack(a, 10, 6)}
assert(next(x) == nil)   -- no elements
x = {unpack(a, 11, 10)}
assert(next(x) == nil)   -- no elements
x,y = unpack(a, 10, 10)
assert(x == 10 and y == nil)
x,y,z = unpack(a, 10, 11)
assert(x == 10 and y == 11 and z == nil)
a,x = unpack{1}
assert(a==1 and x==nil)
a,x = unpack({1,2}, 1, 1)
assert(a==1 and x==nil)
end
