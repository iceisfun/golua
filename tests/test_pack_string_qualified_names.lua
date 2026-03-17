-- string.pack/unpack/packsize error messages should use qualified names
-- (string.pack, string.unpack, string.packsize), matching Lua 5.4 behavior.

local ok, err

-- string.pack
ok, err = pcall(string.pack, "b", 999)
assert(err:find("'string%.pack'"), "expected 'string.pack', got: " .. err)

-- string.pack unsigned overflow
ok, err = pcall(string.pack, "B", -1)
assert(err:find("'string%.pack'"), "expected 'string.pack', got: " .. err)

-- string.pack string contains zeros
ok, err = pcall(string.pack, "z", "a\0b")
assert(err:find("'string%.pack'"), "expected 'string.pack', got: " .. err)

-- string.unpack data too short
ok, err = pcall(string.unpack, "i4", "ab")
assert(err:find("'string%.unpack'"), "expected 'string.unpack', got: " .. err)

-- string.unpack z format
ok, err = pcall(string.unpack, "z", "no null here")
assert(err:find("'string%.unpack'"), "expected 'string.unpack', got: " .. err)

-- string.packsize variable-length
ok, err = pcall(string.packsize, "z")
assert(err:find("'string%.packsize'"), "expected 'string.packsize', got: " .. err)

-- string.packsize format too large
ok, err = pcall(string.packsize, "i16i16i16i16i16i16i16i16i16i16i16i16i16i16i16i16")
-- This may or may not exceed max, just checking format

print("OK")
