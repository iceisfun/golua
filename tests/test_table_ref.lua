-- Test table reference semantics
local t = {}
t.x = 10
local u = t
u.x = 20

assert(t.x == 20)
