-- Test: Empty table constructor hitting EOF produces "unexpected symbol"
-- matching Lua 5.4 behavior

local ok1, err1 = load("local x = {")
assert(not ok1)
assert(string.find(err1, "unexpected symbol near <eof>", 1, true),
       "expected 'unexpected symbol near <eof>' in: " .. tostring(err1))

-- Multi-line with content should still use checkMatch
local ok2, err2 = load("local x = {\n1\n")
assert(not ok2)
assert(string.find(err2, "to close '{' at line 1", 1, true),
       "expected 'to close' in: " .. tostring(err2))

print("OK")
