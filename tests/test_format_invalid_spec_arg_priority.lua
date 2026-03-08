-- string.format should check arg availability before specifier validity
-- When an invalid specifier has no matching arg, Lua 5.4 reports "no value"
-- rather than "invalid conversion"
local ok, err

-- Invalid specifier with no arg: should be "no value" error
ok, err = pcall(string.format, "%z")
assert(err == "bad argument #2 to 'string.format' (no value)",
  "expected 'no value' for %%z without arg, got: " .. tostring(err))

-- %d then %z with only one arg: should be "no value" for arg #3
ok, err = pcall(string.format, "%d%z", 42)
assert(err == "bad argument #3 to 'string.format' (no value)",
  "expected 'no value' for missing arg after %%d%%z, got: " .. tostring(err))

print("OK")
