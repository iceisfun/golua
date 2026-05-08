-- table.pack with nil args: t.n is authoritative; rawlen follows Lua 5.5
-- first-hole semantics for sparse arrays.
local t = table.pack(1, nil, 3)
assert(t.n == 3)
assert(rawlen(t) == 1, "rawlen should be 1, got " .. rawlen(t))
assert(t[1] == 1)
assert(t[2] == nil)
assert(t[3] == 3)

local t2 = table.pack(nil, 1)
assert(t2.n == 2)
assert(rawlen(t2) == 0, "rawlen should be 0, got " .. rawlen(t2))

local t3 = table.pack(nil, nil, nil)
assert(t3.n == 3)
assert(rawlen(t3) == 0, "rawlen should be 0, got " .. rawlen(t3))

print("OK")
