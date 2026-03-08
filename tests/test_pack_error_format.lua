-- string.pack/unpack/packsize error message format matches Lua 5.4
local ok, err

-- Invalid format option: no "bad argument" wrapper
ok, err = pcall(string.pack, "Z", 1)
assert(err:find("invalid format option 'Z'"), "got: " .. err)
assert(not err:find("bad argument"), "should not have 'bad argument' prefix: " .. err)

-- Integral size out of limits: includes actual size
ok, err = pcall(string.pack, "i0", 1)
assert(err:find("integral size %(0%) out of limits %[1,16%]"), "got: " .. err)

ok, err = pcall(string.pack, "i20", 1)
assert(err:find("integral size %(20%) out of limits %[1,16%]"), "got: " .. err)

-- String length does not fit
ok, err = pcall(string.pack, "s1", string.rep("x", 256))
assert(err:find("string length does not fit in given size"), "got: " .. err)

-- unpack invalid format
ok, err = pcall(string.unpack, "Z", "x")
assert(err:find("invalid format option 'Z'"), "unpack got: " .. err)

-- packsize invalid format
ok, err = pcall(string.packsize, "Z")
assert(err:find("invalid format option 'Z'"), "packsize got: " .. err)
