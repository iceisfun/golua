-- Test: nextvar.lua - __pairs metamethod
-- From: nextvar.lua
-- What: Tests that the __pairs metamethod overrides the default pairs behavior to provide a custom iterator.

do
  local a = {}
  do
    local x,y,z = pairs(a)
    assert(type(x) == 'function' and y == a and z == nil)
  end

  local function foo (e,i)
    assert(e == a)
    if i <= 10 then return i+1, i+2 end
  end

  local function foo1 (e,i)
    i = i + 1
    assert(e == a)
    if i <= e.n then return i,a[i] end
  end

  setmetatable(a, {__pairs = function (x) return foo, x, 0 end})

  local i = 0
  for k,v in pairs(a) do
    i = i + 1
    assert(k == i and v == k+1)
  end

  a.n = 5
  a[3] = 30
end
