-- Deeply nested control structures should hit the parser depth limit.
-- Lua 5.4 rejects these via C stack overflow; golua uses a counter.

-- Deeply nested do/end blocks
local s1 = string.rep("do ", 200) .. string.rep("end ", 200)
local f1, e1 = load(s1)
print(f1 == nil)
--> =true

-- Deeply nested while loops
local s2 = string.rep("while true do ", 200) .. string.rep("end ", 200)
local f2, e2 = load(s2)
print(f2 == nil)
--> =true

-- Deeply nested if/then blocks
local s3 = string.rep("if true then ", 200) .. string.rep("end ", 200)
local f3, e3 = load(s3)
print(f3 == nil)
--> =true

-- Deeply nested repeat/until blocks
local s4 = string.rep("repeat ", 200) .. string.rep("until true ", 200)
local f4, e4 = load(s4)
print(f4 == nil)
--> =true
