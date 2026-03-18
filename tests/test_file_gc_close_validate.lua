-- File metatable __gc and __close should validate arguments

-- __gc with no args
local mt = getmetatable(io.stdin)
local ok, err = pcall(mt.__gc)
assert(not ok, "__gc with no args should error")
assert(string.find(err, "FILE%* expected, got no value"), "wrong error for __gc no args: " .. tostring(err))

-- __close with no args
local ok2, err2 = pcall(mt.__close)
assert(not ok2, "__close with no args should error")
assert(string.find(err2, "FILE%* expected, got no value"), "wrong error for __close no args: " .. tostring(err2))

-- __gc with wrong type
local ok3, err3 = pcall(mt.__gc, 42)
assert(not ok3, "__gc with number should error")
assert(string.find(err3, "FILE%* expected, got number"), "wrong error for __gc number: " .. tostring(err3))

-- __close with wrong type
local ok4, err4 = pcall(mt.__close, "hello")
assert(not ok4, "__close with string should error")
assert(string.find(err4, "FILE%* expected, got string"), "wrong error for __close string: " .. tostring(err4))

print("PASS")
