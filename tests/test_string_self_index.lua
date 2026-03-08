-- Test that OP_SELF on strings resolves through __index only,
-- not by direct lookup in the string metatable.

local smt = getmetatable("")

-- Add a custom field directly to the string metatable
smt.custom_field = function(s) return "CUSTOM:" .. s end

-- Field access should return nil (goes through __index which points to string lib)
assert(("test").custom_field == nil, "direct field access should be nil")

-- Method call via OP_SELF should also fail (should go through __index only)
local ok, err = pcall(function() return ("test"):custom_field() end)
assert(not ok, "method call on custom metatable field should fail")

-- Verify real string methods still work via __index
assert(("hello"):upper() == "HELLO", "real string methods should still work")
assert(("hello"):sub(1,3) == "hel", "string.sub should still work")

-- Clean up
smt.custom_field = nil

print("OK")
