-- Test: closure.lua - Closures with generic for control variable
-- From: closure.lua (updated for Lua 5.5)
-- What: Tests closures capturing both numeric index and value variables in generic for.
-- In Lua 5.5, the for-loop control variable (i) is read-only, so we shadow
-- it with a mutable local for closure capture tests.

do
  a = {}
  local t = {"a", "b"}
  for i = 1, #t do
    local ii = i  -- shadow with mutable local
    local k = t[i]
    a[i] = {set = function(x, y) ii=x; k=y end,
            get = function () return ii, k end}
    if i == 2 then break end
  end
  a[1].set(10, 20)
  local r,s = a[2].get()
  assert(r == 2 and s == 'b')
end
