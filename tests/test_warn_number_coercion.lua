-- Test: warn() coerces numbers to strings via tostring
-- warn() accepts strings and numbers, and numbers should be
-- converted to their string representation (using tostring format).

-- Verify warn(42) doesn't error
local ok, err = pcall(warn, 42)
assert(ok == true, "warn(42) should succeed, got: " .. tostring(err))

-- Verify warn(42.0) doesn't error
ok, err = pcall(warn, 42.0)
assert(ok == true, "warn(42.0) should succeed, got: " .. tostring(err))

-- Verify warn with mixed string/number args
ok, err = pcall(warn, "value=", 42)
assert(ok == true, "warn('value=', 42) should succeed")

-- Verify warn with float arg
ok, err = pcall(warn, "pi=", 3.14)
assert(ok == true, "warn('pi=', 3.14) should succeed")

-- Non-string/number args should still error
ok, err = pcall(warn, nil)
assert(ok == false, "warn(nil) should fail")
assert(string.find(err, "string expected"), "warn(nil) error message")

ok, err = pcall(warn, true)
assert(ok == false, "warn(true) should fail")

ok, err = pcall(warn, {})
assert(ok == false, "warn({}) should fail")
