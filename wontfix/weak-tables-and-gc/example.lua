-- Weak tables, __gc finalizers, and collection TIMING.
--
-- golua runs on the Go runtime and uses the Go garbage collector. Lua's own
-- incremental collector, weak-table reclamation guarantees, and the precise
-- moment finalizers run do not map onto Go's GC. Anything that depends on WHEN
-- an unreferenced value is collected is therefore not reproducible against the
-- reference interpreter.

-- Weak-valued table: reference Lua guarantees the entry is gone after a full
-- collection; under the Go GC the timing is not controllable from Lua.
local t = setmetatable({}, {__mode = "v"})
t[1] = {}
t[1] = nil                       -- drop the only strong reference
collectgarbage()                 -- a hint at best; does not force Go's GC
-- Whether next(t) is nil here is timing-dependent under the Go GC and may
-- differ from reference Lua. Do not rely on it.
print("weak value cleared:", next(t) == nil)   -- result is not guaranteed

-- collectgarbage("count") deltas and the exact step/pause behavior of the
-- incremental collector are also not reproducible (golua maps collectgarbage to
-- Go GC hints; see the notes below).
print(math.type(collectgarbage("count")))      --> "float" on both, value differs
