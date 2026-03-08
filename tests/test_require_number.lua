-- Test that require() accepts numbers by coercing to string

-- require(42) should attempt to find module "42", not type-error
local ok, err = pcall(require, 42)
-- It should fail with "module not found" not "string expected"
assert(not ok, "require(42) should fail (module not found)")
assert(not err:find("string expected"), "should not say 'string expected': " .. tostring(err))
assert(err:find("module '42' not found") or err:find("42"), "should try to find module '42': " .. tostring(err))

-- require(1.5) should coerce to "1.5"
ok, err = pcall(require, 1.5)
assert(not ok)
assert(not err:find("string expected"), "should not say 'string expected' for 1.5: " .. tostring(err))

-- require(true) should still error with type error (non-coercible)
ok, err = pcall(require, true)
assert(not ok)
assert(err:find("string expected"), "require(true) should type-error: " .. tostring(err))

-- require(nil) should error
ok, err = pcall(require, nil)
assert(not ok)

print("OK")
