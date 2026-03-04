-- Test: Missing arguments in tail call
-- From: vararg.lua
-- What: Tests that a tail call with fewer arguments than parameters passes nil for missing args.

do
  local function f(a,b,c) return c, b end
  local function g() return f(1,2) end
  local a, b = g()
  assert(a == nil and b == 2)
end
