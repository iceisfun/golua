-- BROKEN: GC finalization timing is non-deterministic under Go's GC.
-- golua delegates garbage collection entirely to Go's runtime.GC() and
-- does not attempt to match C Lua's deterministic collector behavior.
-- Go's GC does not guarantee that all finalizers run within a single
-- GC cycle, so tests depending on exact finalization counts may fail.
-- Guarantees: correctness, eventual finalization, weak table cleanup,
-- no resurrection invariant violations. See README.md.
--
-- Original bug: GC finalizer has off-by-one: the last created object's __gc
-- is not called. With 5 objects, only 4 get finalized.

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
