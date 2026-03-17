-- Test that file method error messages use correct argument numbers.
-- In Lua 5.4, method calls (f:read, f:write, etc.) count self as arg #1,
-- so the first explicit argument is arg #2. The function name shows as '?'
-- because C Lua can't resolve method call names.

local f = assert(io.open(os.tmpname(), "w+"))

-- f:read("x") should say "bad argument #2 to '?' (invalid format)"
local ok, err = pcall(function() f:read("x") end)
assert(not ok, "f:read('x') should error")
assert(string.find(err, "bad argument #2 to '%?'"),
  "read bad format should be arg #2, got: " .. tostring(err))

-- f:write(nil) should say "bad argument #2 to '?' (string expected, got nil)"
ok, err = pcall(function() f:write(nil) end)
assert(not ok, "f:write(nil) should error")
assert(string.find(err, "bad argument #2 to '%?'"),
  "write nil should be arg #2, got: " .. tostring(err))

-- f:seek("bad") should say "bad argument #2 to '?' (invalid option 'bad')"
ok, err = pcall(function() f:seek("bad") end)
assert(not ok, "f:seek('bad') should error")
assert(string.find(err, "bad argument #2 to '%?'"),
  "seek bad whence should be arg #2, got: " .. tostring(err))

-- f:setvbuf("bad") should say "bad argument #2 to '?' (invalid option 'bad')"
ok, err = pcall(function() f:setvbuf("bad") end)
assert(not ok, "f:setvbuf('bad') should error")
assert(string.find(err, "bad argument #2 to '%?'"),
  "setvbuf bad mode should be arg #2, got: " .. tostring(err))

-- f:write(true) should say "bad argument #2 to '?'"
ok, err = pcall(function() f:write(true) end)
assert(not ok, "f:write(true) should error")
assert(string.find(err, "bad argument #2 to '%?'"),
  "write true should be arg #2, got: " .. tostring(err))

-- f:write("ok", nil) should say "bad argument #3 to '?'"
ok, err = pcall(function() f:write("ok", nil) end)
assert(not ok, "f:write('ok', nil) should error")
assert(string.find(err, "bad argument #3 to '%?'"),
  "write second arg nil should be arg #3, got: " .. tostring(err))

f:close()

print("PASS")
