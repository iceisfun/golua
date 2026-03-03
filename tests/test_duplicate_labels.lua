-- Bug: Duplicate labels in the same block are not rejected.
-- Lua 5.4 requires compile-time error for duplicate labels.

-- Test 1: duplicate labels in same block should fail
local f1, err1 = load("::lab:: ::lab::")
assert(f1 == nil, "duplicate labels should fail compilation")
assert(err1:find("label") and err1:find("lab"),
  "error should mention duplicate label: " .. tostring(err1))

-- Test 2: same label name in different blocks is OK
local f2, err2 = load("do ::lab:: end do ::lab:: end")
assert(f2, "same label in different blocks should compile: " .. tostring(err2))

-- Test 3: outer label followed by same label in nested block is rejected
-- (Lua 5.4.8 behavior)
local f3, err3 = load("::lab:: do ::lab:: end")
assert(f3 == nil, "outer then nested duplicate should fail: " .. tostring(err3))
assert(err3:find("label") and err3:find("lab"),
  "error should mention duplicate label: " .. tostring(err3))

print("PASS")
