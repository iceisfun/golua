-- Test: db.lua - Vararg locals
-- From: db.lua
-- What: Tests debug.getlocal/setlocal with negative indices to access vararg values

do
  local function foo (a, ...)
    local t = table.pack(...)
    for i = 1, t.n do
      local n, v = debug.getlocal(1, -i)
      assert(n == "(vararg)" and v == t[i])
    end
    assert(not debug.getlocal(1, -(t.n + 1)))
  end
  foo()
  foo(print)
  foo(200, 3, 4)
end
