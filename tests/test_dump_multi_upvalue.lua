-- Test: loaded dump with 2+ upvalues doesn't crash (nil pointer dereference)
-- When a function has multiple upvalues, all must be initialized.
-- The first upvalue gets _ENV (globals), rest get nil.
local x, y = 1, 2
local f = function() return x, y end
local s = string.dump(f)
local g = load(s)
local a, b = g()
-- First upvalue is _ENV (globals table), second is nil
assert(type(a) == "table", "expected table, got: " .. tostring(a))
assert(b == nil, "expected nil, got: " .. tostring(b))

-- Test with 3 upvalues
local x2, y2, z2 = 10, 20, 30
local f2 = function() return x2, y2, z2 end
local s2 = string.dump(f2)
local g2 = load(s2)
local a2, b2, c2 = g2()
assert(type(a2) == "table", "expected table for first upvalue")
assert(b2 == nil, "expected nil for second upvalue")
assert(c2 == nil, "expected nil for third upvalue")

print("OK")
