-- Test string.dump argument errors.
-- Lua 5.5's str_dump uses a single luaL_argcheck: any arg #1 that is not a
-- Lua function (missing, nil, wrong type, or a C/native function) produces
-- the same "(Lua function expected)" message.

local ok, err = pcall(string.dump)
assert(not ok)
assert(err:find("Lua function expected"), "dump() should say 'Lua function expected', got: " .. tostring(err))

ok, err = pcall(string.dump, nil)
assert(not ok)
assert(err:find("Lua function expected"), "dump(nil) should say 'Lua function expected', got: " .. tostring(err))

ok, err = pcall(string.dump, 42)
assert(not ok)
assert(err:find("Lua function expected"), "dump(42) should say 'Lua function expected', got: " .. tostring(err))

ok, err = pcall(string.dump, "hello")
assert(not ok)
assert(err:find("Lua function expected"), "dump('hello') should say 'Lua function expected', got: " .. tostring(err))

-- A C/native function is also rejected with the same message.
ok, err = pcall(string.dump, print)
assert(not ok)
assert(err:find("Lua function expected"), "dump(print) should say 'Lua function expected', got: " .. tostring(err))

print("OK")
