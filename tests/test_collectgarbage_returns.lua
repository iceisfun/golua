-- Test: collectgarbage("incremental"|"generational") returns the previous
-- mode as a single string value, and the default mode is reported as
-- "generational" on the first switch.  collectgarbage("step") returns
-- false unless the step completes a full GC cycle.

-- Default mode is reported as "generational" on the first switch.
assert(collectgarbage("incremental") == "generational",
  "first switch to incremental should return 'generational'")

-- Subsequent switches return the previous mode.
assert(collectgarbage("generational") == "incremental",
  "switch back to generational should return 'incremental'")
assert(collectgarbage("incremental") == "generational")

-- Single return value, not three.
assert(select("#", collectgarbage("incremental")) == 1,
  "collectgarbage('incremental') must return exactly 1 value")
assert(select("#", collectgarbage("generational")) == 1)

-- step returns a boolean (false unless a full cycle just completed).
local r1 = collectgarbage("step")
assert(type(r1) == "boolean", "step must return boolean, got " .. type(r1))
local r2 = collectgarbage("step", 100)
assert(type(r2) == "boolean")

print("OK")
