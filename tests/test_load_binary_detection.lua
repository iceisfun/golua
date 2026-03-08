-- Test: binary chunk detection checks only first byte (0x1B)
-- Any data starting with \27 should be treated as binary, not just "\27Lua"

-- "\27Not" should be detected as binary and rejected in text mode
local g, err = load("\27Not", "=test", "b")
assert(g == nil, "should fail to load invalid binary")
assert(string.find(err, "bad binary format"), "error should mention bad binary format, got: " .. tostring(err))

-- In text mode, "\27Not" should be rejected as binary, not treated as text
local g2, err2 = load("\27Not", "=test2", "t")
assert(g2 == nil, "should reject binary in text mode")
assert(string.find(err2, "binary chunk"), "error should mention binary chunk, got: " .. tostring(err2))

-- "\27" alone should be treated as binary
local g3, err3 = load("\27", "=test3", "b")
assert(g3 == nil, "should fail to load truncated binary")
assert(string.find(err3, "bad binary format") or string.find(err3, "truncated"),
    "error should mention bad binary format, got: " .. tostring(err3))

print("OK")
