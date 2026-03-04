-- Test: events.lua - __index as function
-- From: events.lua
-- What: Tests __index metamethod as a function that accesses parent table

do
  local t = {}
  local a = {10, 20, 30; x="10", y="20"}
  assert(setmetatable(a, t) == a)
  assert(getmetatable(a) == t)
  assert(setmetatable(a, nil) == a)
  assert(getmetatable(a) == nil)
  assert(setmetatable(a, t) == a)

  local function f (t, i, e)
    assert(not e)
    local p = rawget(t, "parent")
    return (p and p[i]+3), "dummy return"
  end

  t.__index = f

  a.parent = {z=25, x=12, [4] = 24}
  assert(a[1] == 10 and a.z == 28 and a[4] == 27 and a.x == "10")
end
