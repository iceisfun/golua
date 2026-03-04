-- Test: coroutine.lua - Recursive coroutine (factorial via yields)
-- From: coroutine.lua
-- What: Tests recursive function that yields intermediate results

do
  local function pf (n, i)
    coroutine.yield(n)
    pf(n*i, i+1)
  end
  local f = coroutine.wrap(pf)
  local s=1
  for i=1,10 do
    assert(f(1, 1) == s)
    s = s*i
  end
end
