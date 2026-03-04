-- Test: Vararg with math.max
-- From: vararg.lua
-- What: Tests table.unpack expanding an array into math.max arguments.

do
local lim = 20
local a = {}
local i = 1
while i <= lim do a[i] = i; i=i+1 end
local call = function (f, args) return f(table.unpack(args, 1, args.n)) end
assert(call(math.max, a) == lim)
end
