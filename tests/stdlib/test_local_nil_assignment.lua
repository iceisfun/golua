-- Test: Local variable nil assignment
-- From: locals.lua
-- What: Tests that assigning nil to a local parameter returns nil, and that uninitialized locals are nil.

do
  local function f(x) x = nil; return x end
  assert(f(10) == nil)

  local function f() local x; return x end
  assert(f(10) == nil)

  local function f(x) x = nil; local y; return x, y end
  assert(f(10) == nil and select(2, f(20)) == nil)
end
