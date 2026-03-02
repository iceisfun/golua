-- Test: coroutine.lua - Coroutine x for loop (permutations)
-- From: coroutine.lua
-- What: Tests generating all permutations using coroutines with for-in loop

do
  local function all (a, n, k)
    if k == 0 then coroutine.yield(a)
    else
      for i=1,n do
        a[k] = i
        all(a, n, k-1)
      end
    end
  end

  local a = 0
  for t in coroutine.wrap(function () all({}, 5, 4) end) do
    a = a+1
  end
  assert(a == 5^4)
end
