-- string.pack error messages should use short name 'pack'
-- matching Lua 5.4 behavior
local ok, err

ok, err = pcall(string.pack, "z", nil)
assert(err == "bad argument #2 to 'pack' (string expected, got nil)",
  "expected 'pack' in error, got: " .. tostring(err))

print("OK")
