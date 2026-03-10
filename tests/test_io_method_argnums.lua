-- Test that file method error messages use correct argument numbers.
-- In Lua 5.4, method calls (f:read, f:write, etc.) count arguments
-- starting at 1 for the first arg AFTER self. The self parameter
-- is not counted in the argument number.

local f = assert(io.open(os.tmpname(), "w+"))

-- f:read("x") should say "bad argument #1 to 'read' (invalid format)"
local ok, err = pcall(function() f:read("x") end)
assert(not ok, "f:read('x') should error")
assert(string.find(err, "bad argument #1 to 'read'"),
  "read bad format should be arg #1, got: " .. tostring(err))
assert(not string.find(err, "bad self"),
  "read bad format should NOT say 'bad self', got: " .. tostring(err))

-- f:write(nil) should say "bad argument #1 to 'write' (string expected, got nil)"
ok, err = pcall(function() f:write(nil) end)
assert(not ok, "f:write(nil) should error")
assert(string.find(err, "bad argument #1 to 'write'"),
  "write nil should be arg #1, got: " .. tostring(err))
assert(not string.find(err, "bad self"),
  "write nil should NOT say 'bad self', got: " .. tostring(err))

-- f:seek("bad") should say "bad argument #1 to 'seek' (invalid option 'bad')"
ok, err = pcall(function() f:seek("bad") end)
assert(not ok, "f:seek('bad') should error")
assert(string.find(err, "bad argument #1 to 'seek'"),
  "seek bad whence should be arg #1, got: " .. tostring(err))

-- f:setvbuf("bad") should say "bad argument #1 to 'setvbuf' (invalid option 'bad')"
ok, err = pcall(function() f:setvbuf("bad") end)
assert(not ok, "f:setvbuf('bad') should error")
assert(string.find(err, "bad argument #1 to 'setvbuf'"),
  "setvbuf bad mode should be arg #1, got: " .. tostring(err))

-- f:write(true) should say "bad argument #1 to 'write'"
ok, err = pcall(function() f:write(true) end)
assert(not ok, "f:write(true) should error")
assert(string.find(err, "bad argument #1 to 'write'"),
  "write true should be arg #1, got: " .. tostring(err))
assert(not string.find(err, "bad self"),
  "write true should NOT say 'bad self', got: " .. tostring(err))

-- f:write("ok", nil) should say "bad argument #2 to 'write'"
ok, err = pcall(function() f:write("ok", nil) end)
assert(not ok, "f:write('ok', nil) should error")
assert(string.find(err, "bad argument #2 to 'write'"),
  "write second arg nil should be arg #2, got: " .. tostring(err))

f:close()

print("PASS")
