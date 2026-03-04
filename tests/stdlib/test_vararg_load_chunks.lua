-- Test: Varargs for main chunks via load
-- From: vararg.lua
-- What: Tests that load creates a vararg function that receives arguments.

do
local f = load[[ return {...} ]]
local x = f(2,3)
assert(x[1] == 2 and x[2] == 3 and x[3] == undef)

f = load[[
  local x = {...}
  for i=1,select('#', ...) do assert(x[i] == select(i, ...)) end
  assert(x[select('#', ...)+1] == undef)
  return true
]]

assert(f("a", "b", nil, {}, assert))
assert(f())
end
