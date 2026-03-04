-- Test: db.lua - Upvalue access
-- From: db.lua
-- What: Tests debug.getupvalue and debug.setupvalue

do
  local a,b,c = 1,2,3
  local function foo1 (a) b = a; return c end
  local function foo2 (x) a = x; return c+b end
  assert(not debug.getupvalue(foo1, 3))
  assert(debug.setupvalue(foo1, 1, "xuxu") == "b")
  assert(({debug.getupvalue(foo2, 3)})[2] == "xuxu")
end
