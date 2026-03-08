-- Test that string.format invalid conversion includes full specifier

local ok, err = pcall(string.format, "%10.5r", 42)
assert(not ok)
assert(err:find("'%%10.5r'"), "should include full spec '%10.5r', got: " .. err)

ok, err = pcall(string.format, "%-10r", 42)
assert(not ok)
assert(err:find("'%%%-10r'"), "should include full spec '%-10r', got: " .. err)

ok, err = pcall(string.format, "%.5r", 42)
assert(not ok)
assert(err:find("'%%%.5r'"), "should include full spec '%.5r', got: " .. err)

-- Simple case should still work
ok, err = pcall(string.format, "%r", 42)
assert(not ok)
assert(err:find("'%%r'"), "simple case should say '%r', got: " .. err)

print("OK")
