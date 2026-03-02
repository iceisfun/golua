-- Test: closure.lua - Closures with generic for control variable
-- From: closure.lua
-- What: Tests closures capturing both numeric index and value variables in generic for

do
  a = {}
  local t = {"a", "b"}
  for i = 1, #t do
    local k = t[i]
    a[i] = {set = function(x, y) i=x; k=y end,
            get = function () return i, k end}
    if i == 2 then break end
  end
  a[1].set(10, 20)
  local r,s = a[2].get()
  assert(r == 2 and s == 'b')
end
