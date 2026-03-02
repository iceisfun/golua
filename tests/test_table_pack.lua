-- Test: table.pack
-- From: sort.lua
-- What: Tests table.pack with zero, one, and multiple (including nil) arguments, verifying the .n field is set correctly.

do
local a = table.pack()
assert(a[1] == undef and a.n == 0)

a = table.pack(table)
assert(a[1] == table and a.n == 1)

a = table.pack(nil, nil, nil, nil)
assert(a[1] == nil and a.n == 4)
end
