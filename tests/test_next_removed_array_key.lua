-- Test: next() should not error on removed integer key at array boundary
-- Bug: when array shrinks after setting last element to nil, next(t, key)
-- returns "invalid key" instead of proceeding to hash or returning nil

local t = {[1]="a", [2]="b", [3]="c", [4]="d"}
t[4] = nil

-- next(t, 4) should return nil (end of iteration), not error
local ok, k, v = pcall(next, t, 4)
assert(ok, "next(t, 4) should not error after t[4]=nil, got: " .. tostring(k))
assert(k == nil, "expected nil key, got: " .. tostring(k))

-- Also test removing a middle-boundary element that causes shrink
local t2 = {[1]="a", [2]="b", [3]="c"}
t2[3] = nil
local ok2, k2, v2 = pcall(next, t2, 3)
assert(ok2, "next(t2, 3) should not error after t2[3]=nil, got: " .. tostring(k2))
assert(k2 == nil, "expected nil key from next(t2, 3), got: " .. tostring(k2))

-- Test with hash entries that should be returned after removed array key
local t3 = {[1]="a", [2]="b", [3]="c"}
t3.x = "hello"
t3[3] = nil
local ok3, k3, v3 = pcall(next, t3, 3)
assert(ok3, "next(t3, 3) should not error, got: " .. tostring(k3))
-- Should get the hash entry "x"
assert(k3 == "x", "expected key 'x', got: " .. tostring(k3))
assert(v3 == "hello", "expected value 'hello', got: " .. tostring(v3))

print("PASS")
