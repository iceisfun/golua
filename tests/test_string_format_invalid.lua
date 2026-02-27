-- Bug: Invalid format specifiers silently pass through instead of erroring.
-- Lua 5.4 raises "invalid conversion '%z'" for unknown format characters.

local ok1, err1 = pcall(string.format, "%z", 42)
assert(not ok1, "%z should be invalid format")
assert(tostring(err1):find("invalid"),
  "error should mention 'invalid', got: " .. tostring(err1))

local ok2, err2 = pcall(string.format, "%r", 42)
assert(not ok2, "%r should be invalid format")

local ok3, err3 = pcall(string.format, "%n", 42)
assert(not ok3, "%n should be invalid format")

-- Valid formats should still work
assert(string.format("%d", 42) == "42", "%d should work")
assert(string.format("%s", "hi") == "hi", "%s should work")
assert(string.format("%%") == "%", "%% should work")

print("PASS")
