-- BUG: String library error messages use short function names instead of
-- fully-qualified names. Reference Lua 5.4 reports e.g. 'string.format',
-- GoLua reports just 'format'.

-- Test with string.format
local ok, err = pcall(string.format, "%d", nil)
assert(not ok)
assert(err:find("'string.format'", 1, true), "expected 'string.format' in error, got: " .. err)

-- Test with string.len
ok, err = pcall(string.len, {})
assert(not ok)
assert(err:find("'string.len'", 1, true), "expected 'string.len' in error, got: " .. err)

-- Test with string.sub
ok, err = pcall(string.sub, {}, 1)
assert(not ok)
assert(err:find("'string.sub'", 1, true), "expected 'string.sub' in error, got: " .. err)

-- Test with string.find
ok, err = pcall(string.find, "hello", "hello", "x")
assert(not ok)
assert(err:find("'string.find'", 1, true), "expected 'string.find' in error, got: " .. err)

-- Test with string.char
ok, err = pcall(string.char, "hello")
assert(not ok)
assert(err:find("'string.char'", 1, true), "expected 'string.char' in error, got: " .. err)

-- Test with string.byte
ok, err = pcall(string.byte, {})
assert(not ok)
assert(err:find("'string.byte'", 1, true), "expected 'string.byte' in error, got: " .. err)

-- Test with string.gsub
ok, err = pcall(string.gsub, "hello", "hello", "x", "y")
assert(not ok)
assert(err:find("'string.gsub'", 1, true), "expected 'string.gsub' in error, got: " .. err)

-- Test with string.rep
ok, err = pcall(string.rep, {}, 3)
assert(not ok)
assert(err:find("'string.rep'", 1, true), "expected 'string.rep' in error, got: " .. err)

print("PASS")
