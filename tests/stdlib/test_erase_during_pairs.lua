-- Test: nextvar.lua - Erasing values during pairs iteration
-- From: nextvar.lua
-- What: Tests deleting table entries during pairs iteration, verifying that each key-value pair is visited exactly once.

do
  local t = {[{1}] = 1, [{2}] = 2, [string.rep("x ", 4)] = 3,
             [100.3] = 4, [4] = 5}

  local n = 0
  for k, v in pairs( t ) do
    n = n+1
    assert(t[k] == v)
    t[k] = undef
    collectgarbage()
    assert(t[k] == undef)
  end
  assert(n == 5)
end
