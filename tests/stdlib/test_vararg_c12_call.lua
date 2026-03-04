-- Test: c12 function and call helper
-- From: vararg.lua
-- What: Tests exact argument matching, table.unpack for indirect calling, and nested call().

do
local function c12 (...)
  assert(arg == _G.arg)
  local x = {...}; x.n = #x
  local res = (x.n==2 and x[1] == 1 and x[2] == 2)
  if res then res = 55 end
  return res, 2
end

local function vararg (...) return {n = select('#', ...), ...} end
local call = function (f, args) return f(table.unpack(args, 1, args.n)) end

assert(c12(1,2)==55)
local a,b = assert(call(c12, {1,2}))
assert(a == 55 and b == 2)
a = call(c12, {1,2;n=2})
assert(a == 55 and b == 2)
a = call(c12, {1,2;n=1})
assert(not a)
assert(c12(1,2,3) == false)
local a = vararg(call(next, {_G,nil;n=2}))
local b,c = next(_G)
assert(a[1] == b and a[2] == c and a.n == 2)
a = vararg(call(call, {c12, {1,2}}))
assert(a.n == 2 and a[1] == 55 and a[2] == 2)
a = call(print, {'+'})
assert(a == nil)
end
