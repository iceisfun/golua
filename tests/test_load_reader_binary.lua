-- Test: reader-based load handles binary data correctly

-- Bug 3: Reader returning invalid binary data should give proper error
local badBinCalled = false
local g, err = load(function()
    if not badBinCalled then badBinCalled = true; return "\27NotBinary" end
end, "=test")
assert(g == nil, "should fail to load invalid binary from reader")
assert(string.find(err, "bad binary format"),
    "error should mention bad binary format, got: " .. tostring(err))

-- Bug 4: Reader returning binary data with mode "t" should be rejected
local g2, err2 = load(function() return "\27Lua..." end, "=test2", "t")
assert(g2 == nil, "should reject binary from reader in text mode")
assert(string.find(err2, "binary chunk"),
    "error should mention binary chunk, got: " .. tostring(err2))

-- Valid binary dump via reader should still work (single chunk)
local f = function() return 42 end
local s = string.dump(f)
local called = false
local g3 = load(function()
    if not called then called = true; return s end
end)
assert(g3 ~= nil, "should load valid binary from reader")
assert(g3() == 42, "loaded function should return 42")

-- Multi-chunk binary reader: binary dump split across multiple reads
local s2 = string.dump(function() return 99 end)
local pos = 0
local g4 = load(function()
  pos = pos + 1
  if pos == 1 then return s2:sub(1, 10) end
  if pos == 2 then return s2:sub(11) end
  return nil
end)
assert(g4, "multi-chunk binary load should work")
assert(g4() == 99, "loaded function should return 99")

-- Non-string source name should error
local ok, err3 = pcall(load, "return 1", false)
assert(not ok, "load with boolean source name should error")
assert(string.find(err3, "bad argument #2"), "expected bad argument #2, got: " .. tostring(err3))

local ok2, err4 = pcall(load, "return 1", {})
assert(not ok2, "load with table source name should error")

print("OK")
