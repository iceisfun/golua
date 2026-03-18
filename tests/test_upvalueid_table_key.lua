-- Lightuserdata values from debug.upvalueid should work as table keys
-- and be visible via next() and pairs().

local function outer()
  local x = 1
  return function() return x end
end
local f = outer()
local id = debug.upvalueid(f, 1)

-- Use upvalueid as a table key
local t = {[id] = "ok"}

-- Direct access works
assert(t[id] == "ok", "direct access failed")

-- next() should find the key
local k, v = next(t)
assert(type(k) == "userdata", "expected userdata key, got " .. type(k))
assert(rawequal(k, id), "next() key not equal to upvalueid")
assert(v == "ok", "next() value should be 'ok'")

-- pairs() should iterate over it
local count = 0
for pk, pv in pairs(t) do
  assert(rawequal(pk, id), "pairs key not equal to upvalueid")
  assert(pv == "ok")
  count = count + 1
end
assert(count == 1, "expected 1 iteration, got " .. count)

print("OK")
