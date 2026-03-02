-- Test: Basic vararg collection and counting
-- From: vararg.lua
-- What: Tests that variadic arguments are correctly collected using select('#', ...) and {...}.

do
local function f (a, ...)
  local x = {n = select('#', ...), ...}
  for i = 1, x.n do assert(a[i] == x[i]) end
  return x.n
end

local function vararg (...) return {n = select('#', ...), ...} end

assert(f() == 0)
assert(f({1,2,3}, 1, 2, 3) == 3)
assert(f({"alo", nil, 45, f, nil}, "alo", nil, 45, f, nil) == 5)

assert(vararg().n == 0)
assert(vararg(nil, nil).n == 2)
end
