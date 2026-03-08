-- string.pack error messages should use qualified name 'string.pack'
-- when called directly (not from Lua code wrapper)
local ok, err

ok, err = pcall(string.pack, "z", nil)
assert(err == "bad argument #2 to 'string.pack' (string expected, got nil)",
  "expected 'string.pack' in error, got: " .. tostring(err))

print("OK")
