-- Test: Global table of loaded chunks (debug.getupvalue)
-- From: locals.lua
-- What: Tests that loaded chunks have _ENV as their first upvalue, and that a custom environment table can be passed to load.

do
  local function getenv (f)
    local a,b = debug.getupvalue(f, 1)
    assert(a == '_ENV')
    return b
  end

  assert(getenv(load"a=3") == _G)
  local c = {}; local f = load("a = 3", nil, nil, c)
  assert(getenv(f) == c)
  assert(c.a == nil)
  f()
  assert(c.a == 3)
end
