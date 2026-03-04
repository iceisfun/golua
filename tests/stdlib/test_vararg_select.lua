-- Test: select with table.unpack and negative indices
-- From: vararg.lua
-- What: Tests select with positive indices, no arguments, and negative indices.

do
a = {select(3, table.unpack{10,20,30,40})}
assert(#a == 2 and a[1] == 30 and a[2] == 40)
a = {select(1)}
assert(next(a) == nil)
a = {select(-1, 3, 5, 7)}
assert(a[1] == 7 and a[2] == undef)
a = {select(-2, 3, 5, 7)}
assert(a[1] == 5 and a[2] == 7 and a[3] == undef)
pcall(select, 10000)
pcall(select, -10000)
end
