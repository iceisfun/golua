-- test_gc_finalizer_count: all queued __gc handlers run after collectgarbage().
--
-- Two collectgarbage() cycles (queue finalizers, then drain) reliably finalize
-- all five short-lived objects. Previously filed as flaky-broken; now stable
-- after the coroutine-VM stack/retBuf release fix removed the spurious pins
-- that masked finalization.
--
-- Original bug: GC finalizer off-by-one — the last created object's __gc was
-- not called (with 5 objects only 4 got finalized).

local count = 0
for i = 1, 5 do
  setmetatable({id = i}, {
    __gc = function(self)
      count = count + 1
    end
  })
end

collectgarbage()
collectgarbage()

assert(count == 5, "all 5 objects should be finalized, got " .. count)

print("PASS")
