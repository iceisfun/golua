-- Test: nextvar.lua - next x GC of deleted keys
-- From: nextvar.lua
-- What: Tests that next correctly handles GC collection of dead keys during iteration (bug fix from 5.4.1), using coroutines to interleave iteration with GC.

do
  print("testing next x GC of deleted keys")
  -- bug in 5.4.1
  local co = coroutine.wrap(function (t)
    for k, v in pairs(t) do
        local k1 = next(t)    -- all previous keys were deleted
        assert(k == k1)       -- current key is the first in the table
        t[k] = nil
        local expected = (type(k) == "table" and k[1] or
                          type(k) == "function" and k() or
                          string.sub(k, 1, 1))
        assert(expected == v)
        coroutine.yield(v)
    end
  end)
  local t = {}
  t[{1}] = 1    -- add several unanchored, collectable keys
  t[{2}] = 2
  t[string.rep("a", 50)] = "a"    -- long string
  t[string.rep("b", 50)] = "b"
  t[{3}] = 3
  t[string.rep("c", 10)] = "c"    -- short string
  t[function () return 10 end] = 10
  local count = 7
  while co(t) do
    collectgarbage("collect")   -- collect dead keys
    count = count - 1
  end
  assert(count == 0 and next(t) == nil)    -- traversed the whole table
end
