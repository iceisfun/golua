-- Bug: Multiple assignment reverses upvalue registration order
-- When upvalues are first referenced as targets of a multiple assignment,
-- they are registered in reverse order compared to Lua 5.4.
-- This causes debug.getupvalue to return upvalues in the wrong order
-- and debug.upvaluejoin to operate on the wrong upvalue index.

local a = 1
local b = 2
local c = 3

-- Multiple assignment target: a, b, c = ...
local function setter() a, b, c = 10, 20, 30 end

-- Lua 5.4 registers upvalues in source order: a, b, c
local n1 = debug.getupvalue(setter, 1)
local n2 = debug.getupvalue(setter, 2)
local n3 = debug.getupvalue(setter, 3)

assert(n1 == "a", "upvalue 1 should be 'a', got '" .. tostring(n1) .. "'")
assert(n2 == "b", "upvalue 2 should be 'b', got '" .. tostring(n2) .. "'")
assert(n3 == "c", "upvalue 3 should be 'c', got '" .. tostring(n3) .. "'")

-- Also verify with two-target assignment
local x = 1
local y = 2
local function setter2() x, y = 10, 20 end
local nx = debug.getupvalue(setter2, 1)
local ny = debug.getupvalue(setter2, 2)
assert(nx == "x", "upvalue 1 should be 'x', got '" .. tostring(nx) .. "'")
assert(ny == "y", "upvalue 2 should be 'y', got '" .. tostring(ny) .. "'")

print("PASSED")
