-- Test that file method error messages match Lua 5.4's luaL_argerror behavior.
-- Method calls (f:write) resolve the name from bytecode and decrement arg# for self.

local f = assert(io.open(os.tmpname(), "w+"))

-- f:read("x") via method syntax → arg #1, name 'read'
local ok, err = pcall(function() f:read("x") end)
assert(not ok, "f:read('x') should error")
assert(string.find(err, "bad argument #1 to 'read'"),
  "read bad format should be arg #1 to 'read', got: " .. tostring(err))

-- f:write(nil) via method syntax → arg #1, name 'write'
ok, err = pcall(function() f:write(nil) end)
assert(not ok, "f:write(nil) should error")
assert(string.find(err, "bad argument #1 to 'write'"),
  "write nil should be arg #1 to 'write', got: " .. tostring(err))

-- f:seek("bad") via method syntax → arg #1, name 'seek'
ok, err = pcall(function() f:seek("bad") end)
assert(not ok, "f:seek('bad') should error")
assert(string.find(err, "bad argument #1 to 'seek'"),
  "seek bad whence should be arg #1 to 'seek', got: " .. tostring(err))

-- f:setvbuf("bad") via method syntax → arg #1, name 'setvbuf'
ok, err = pcall(function() f:setvbuf("bad") end)
assert(not ok, "f:setvbuf('bad') should error")
assert(string.find(err, "bad argument #1 to 'setvbuf'"),
  "setvbuf bad mode should be arg #1 to 'setvbuf', got: " .. tostring(err))

-- f:write("ok", nil) via method syntax → arg #2, name 'write'
ok, err = pcall(function() f:write("ok", nil) end)
assert(not ok, "f:write('ok', nil) should error")
assert(string.find(err, "bad argument #2 to 'write'"),
  "write second arg nil should be arg #2 to 'write', got: " .. tostring(err))

f:close()

print("PASS")
