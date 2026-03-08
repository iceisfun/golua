-- Test: null byte handling in error messages matches Lua 5.4

-- Bug 1: load("\0") should produce "unexpected symbol" without "near" clause.
-- In C Lua, the null byte terminates the near-token buffer, so it's omitted.
local ok1, err1 = load("\0")
assert(not ok1, "load should fail")
assert(err1:find("unexpected symbol"), "should say 'unexpected symbol', got: " .. tostring(err1))
assert(not err1:find("near"), "should NOT contain 'near' for null byte, got: " .. tostring(err1))

-- Bug 2: null bytes in chunk names should truncate at the null byte.
-- load("x\0y = 1") uses the source text as the chunk name, so
-- the [string "..."] prefix should show "x" not "xy".
local ok2, err2 = load("x\0y = 1")
assert(not ok2, "load should fail")
assert(err2:find('[string "x"]', 1, true),
    'expected [string "x"] in error (truncate at null), got: ' .. tostring(err2))

print("OK")
