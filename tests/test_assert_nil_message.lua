-- assert(false, nil) should raise nil as the error value,
-- which is replaced by "<no error object>" per Lua 5.5 semantics.

-- Test 1: assert(false, nil) - Lua 5.5: nil error replaced by string
local ok1, err1 = pcall(assert, false, nil)
assert(not ok1, "assert(false, nil) should fail")
-- Lua 5.5: nil error object is replaced by "<no error object>"
assert(err1 == "<no error object>", "assert(false, nil) error should be '<no error object>', got " .. type(err1) .. ": " .. tostring(err1))

-- Test 2: assert(false) with no 2nd arg should use default message
local ok2, err2 = pcall(assert, false)
assert(not ok2, "assert(false) should fail")
assert(err2 == "assertion failed!", "assert(false) should say 'assertion failed!', got: " .. tostring(err2))

-- Test 3: assert(false, "custom") should use custom message (baseline)
local ok3, err3 = pcall(assert, false, "custom")
assert(not ok3)
assert(err3 == "custom", "got: " .. tostring(err3))

-- Test 4: assert(false, 42) should pass through number
local ok4, err4 = pcall(assert, false, 42)
assert(not ok4)
assert(err4 == 42, "got: " .. tostring(err4))

print("PASS")
