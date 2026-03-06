-- BUG: string.format error messages for invalid conversions differ from Lua 5.4.
--
-- Issue 1: Invalid conversion character missing " to 'format'" suffix.
-- Reference: "invalid conversion '%w' to 'format'"
-- GoLua:     "invalid conversion '%w'"
--
-- Issue 2: Invalid flag combination uses wrong error format.
-- Reference: "invalid conversion specification: '%#d'"
-- GoLua:     "invalid conversion '%#d'"

-- Test 1: invalid conversion character should include " to 'format'"
local ok, err = pcall(string.format, "%w", 42)
assert(not ok)
assert(err:find("to 'string.format'", 1, true),
  "expected \" to 'string.format'\" suffix in error, got: " .. err)

-- Test 2: invalid flag combo should say "invalid conversion specification:"
ok, err = pcall(string.format, "%#d", 42)
assert(not ok)
assert(err:find("invalid conversion specification:", 1, true),
  "expected 'invalid conversion specification:', got: " .. err)

-- Test 3: another invalid flag combo
ok, err = pcall(string.format, "%+u", 42)
assert(not ok)
assert(err:find("invalid conversion specification:", 1, true),
  "expected 'invalid conversion specification:', got: " .. err)

-- Test 4: space flag with %o
ok, err = pcall(string.format, "% o", 42)
assert(not ok)
assert(err:find("invalid conversion specification:", 1, true),
  "expected 'invalid conversion specification:', got: " .. err)

-- Test 5: 0 flag with %s
ok, err = pcall(string.format, "%0s", "x")
assert(not ok)
assert(err:find("invalid conversion specification:", 1, true),
  "expected 'invalid conversion specification:', got: " .. err)

print("PASS")
