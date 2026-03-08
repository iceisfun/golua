-- table.pack with nil args should still have correct rawlen
local t = table.pack(1, nil, 3)
assert(t.n == 3)
assert(rawlen(t) == 3, "rawlen should be 3, got " .. rawlen(t))
assert(t[1] == 1)
assert(t[2] == nil)
assert(t[3] == 3)

local t2 = table.pack(nil, 1)
assert(t2.n == 2)
assert(rawlen(t2) == 2, "rawlen should be 2, got " .. rawlen(t2))

local t3 = table.pack(nil, nil, nil)
assert(t3.n == 3)
-- rawlen of all-nil is 0 in both Lua 5.4 and GoLua (trailing nils are trimmed)
assert(rawlen(t3) == 0, "rawlen should be 0, got " .. rawlen(t3))

print("OK")
