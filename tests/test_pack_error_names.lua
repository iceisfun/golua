-- Test that pack/unpack/packsize error messages use short function names

-- string.pack overflow
local ok, err = pcall(string.pack, "b", 1000)
assert(not ok)
assert(err:find("to 'pack'"), "pack error should say 'pack': " .. tostring(err))

-- string.unpack too short
ok, err = pcall(string.unpack, "i4", "ab")
assert(not ok)
assert(err:find("to 'unpack'"), "unpack error should say 'unpack': " .. tostring(err))

-- string.packsize variable length
ok, err = pcall(string.packsize, "s1")
assert(not ok)
assert(err:find("to 'packsize'"), "packsize error should say 'packsize': " .. tostring(err))

-- Also test via method-style call
ok, err = pcall(string.pack, "i1", 128)
assert(not ok)
assert(err:find("to 'pack'"), "pack overflow should say 'pack': " .. tostring(err))

ok, err = pcall(string.unpack, "i4", "x", 100)
assert(not ok)
assert(err:find("to 'unpack'"), "unpack offset should say 'unpack': " .. tostring(err))

print("OK")
