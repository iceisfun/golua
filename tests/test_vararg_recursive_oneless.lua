-- Test: New-style vararg with recursive function and oneless
-- From: vararg.lua
-- What: Tests recursive vararg functions and the oneless helper that drops the first argument.

do
local function oneless (a, ...) return ... end

function f (n, a, ...)
  local b
  assert(arg == _G.arg)
  if n == 0 then
    local b, c, d = ...
    return a, b, c, d, oneless(oneless(oneless(...)))
  else
    n, b, a = n-1, ..., a
    assert(b == ...)
    return f(n, a, ...)
  end
end

a,b,c,d,e = assert(f(10,5,4,3,2,1))
assert(a==5 and b==4 and c==3 and d==2 and e==1)

a,b,c,d,e = f(4)
assert(a==nil and b==nil and c==nil and d==nil and e==nil)
end
