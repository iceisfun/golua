-- Bug: next() should NOT coerce float keys to integers.
-- In Lua 5.4, next(t, 1.0) errors with "invalid key to 'next'"
-- when the table has integer key 1 (stored in array part).
-- GoLua incorrectly coerces 1.0 to 1 and finds the key.

-- Case 1: array-only table
local t1 = {10, 20, 30}
local ok1, err1 = pcall(next, t1, 1.0)
assert(ok1 == false, "next(array, 1.0) should fail, got success")
assert(err1:find("invalid key"), "expected 'invalid key' error, got: " .. tostring(err1))

-- Case 2: mixed table
local t2 = {10, a=1}
local ok2, err2 = pcall(next, t2, 1.0)
assert(ok2 == false, "next(mixed, 1.0) should fail, got success")

-- Case 3: table with key 0, next with 0.0
local t3 = {}
t3[0] = "zero"
local ok3, err3 = pcall(next, t3, 0.0)
assert(ok3 == false, "next(t, 0.0) should fail even when t[0] exists")

-- Case 4: non-integer float key SHOULD work
local t4 = {}
t4[1.5] = "yes"
local k, v = next(t4, 1.5)
-- 1.5 is a real float key, so next should succeed (and return nil since it's the only key)

-- Case 5: 2.0 on table with integer key 2
local t5 = {10, 20, 30}
local ok5, err5 = pcall(next, t5, 2.0)
assert(ok5 == false, "next(t, 2.0) should fail for integer key 2")

print("PASSED")
