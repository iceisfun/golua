-- Test: reader-based load handles binary data correctly

-- Bug 3: Reader returning invalid binary data should give proper error
local g, err = load(function() return "\27NotBinary" end, "=test")
assert(g == nil, "should fail to load invalid binary from reader")
assert(string.find(err, "bad binary format"),
    "error should mention bad binary format, got: " .. tostring(err))

-- Bug 4: Reader returning binary data with mode "t" should be rejected
local g2, err2 = load(function() return "\27Lua..." end, "=test2", "t")
assert(g2 == nil, "should reject binary from reader in text mode")
assert(string.find(err2, "binary chunk"),
    "error should mention binary chunk, got: " .. tostring(err2))

-- Valid binary dump via reader should still work
local f = function() return 42 end
local s = string.dump(f)
local called = false
local g3 = load(function()
    if not called then called = true; return s end
end)
assert(g3 ~= nil, "should load valid binary from reader")
assert(g3() == 42, "loaded function should return 42")

print("OK")
