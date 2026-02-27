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

-- Test 3: duplicate labels in nested block vs outer block
-- Inner block shadowing is OK in Lua 5.4
local f3, err3 = load("::lab:: do ::lab:: end")
assert(f3, "nested duplicate label should compile: " .. tostring(err3))

print("PASS")
