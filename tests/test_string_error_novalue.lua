-- BUG: GoLua reports "got nil" for missing arguments instead of "got no value".
-- Reference Lua 5.4 distinguishes between an explicit nil argument and a missing
-- argument (stack position beyond the top).

-- Missing argument (no value)
local ok, err = pcall(string.len)
assert(not ok)
assert(err:find("got no value", 1, true), "expected 'got no value' for missing arg, got: " .. err)

-- Explicit nil argument (should say "got nil")
ok, err = pcall(string.len, nil)
assert(not ok)
assert(err:find("got nil", 1, true), "expected 'got nil' for nil arg, got: " .. err)

-- Verify both say different things
local _, err_missing = pcall(string.len)
local _, err_nil = pcall(string.len, nil)
assert(err_missing ~= err_nil, "errors for missing arg and nil arg should differ")

print("PASS")
