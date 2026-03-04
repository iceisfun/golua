-- Test: calls.lua - Nested field function definitions
-- From: calls.lua
-- What: Tests defining functions on nested table fields (a.b.c.f1)

do
  local a = {b={c={}}}
  function a.b.c.f1 (x) return x+1 end
  function a.b.c:f2 (x,y) self[x] = y end
  assert(a.b.c.f1(4) == 5)
  a.b.c:f2('k', 12); assert(a.b.c.k == 12)
end
