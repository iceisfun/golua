-- string.format invalid conversion should say "to 'format'" not "to 'string.format'"
local ok, err = pcall(string.format, "%z", 42)
assert(err == "invalid conversion '%z' to 'format'",
  "expected \"to 'format'\", got: " .. tostring(err))

ok, err = pcall(string.format, "%v", 42)
assert(err == "invalid conversion '%v' to 'format'",
  "expected \"to 'format'\", got: " .. tostring(err))

ok, err = pcall(string.format, "%*", 42)
assert(err == "invalid conversion '%*' to 'format'",
  "expected \"to 'format'\", got: " .. tostring(err))

print("OK")
