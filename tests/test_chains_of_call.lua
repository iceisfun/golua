-- Test: calls.lua - Chains of __call
-- From: calls.lua
-- What: Tests building a chain of __call metamethods and invoking through them

do
  local N = 20
  local u = table.pack
  for i = 1, N do
    u = setmetatable({i}, {__call = u})
  end
  local Res = u("a", "b", "c")
  assert(Res.n == N + 3)
  for i = 1, N do
    assert(Res[i][1] == i)
  end
  assert(Res[N + 1] == "a" and Res[N + 2] == "b" and Res[N + 3] == "c")
end
