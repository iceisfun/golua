-- Bug: debug.upvalueid errors on out-of-range index instead of returning nil
-- Lua 5.4 returns nil for out-of-range upvalue indices.
-- GoLua raises "invalid upvalue index" error instead.

-- Function with no upvalues
local function noup() return 1 end
local result = debug.upvalueid(noup, 1)
assert(result == nil, "upvalueid on function with no upvalues should return nil, got " .. tostring(result))

-- Function with one upvalue, index 2 (out of range)
local function has_one()
  local x = 1
  return function() return x end
end
local f = has_one()

-- Valid index
local valid = debug.upvalueid(f, 1)
assert(type(valid) == "userdata", "upvalueid with valid index should return userdata")

-- Out of range (too high)
local r2 = debug.upvalueid(f, 2)
assert(r2 == nil, "upvalueid with index 2 on 1-upvalue function should return nil, got " .. tostring(r2))

-- Out of range (zero)
local r0 = debug.upvalueid(f, 0)
assert(r0 == nil, "upvalueid with index 0 should return nil, got " .. tostring(r0))

print("PASSED")
