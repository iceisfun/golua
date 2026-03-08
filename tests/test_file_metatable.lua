-- Test that file handles have a visible metatable
-- In Lua 5.4, getmetatable(io.stdin) returns the file handle metatable

-- Test 1: getmetatable returns a table for file handles
local mt = getmetatable(io.stdin)
assert(type(mt) == "table", "getmetatable(io.stdin) should return a table, got " .. type(mt))

-- Test 2: All standard handles share the same metatable
local mt2 = getmetatable(io.stdout)
local mt3 = getmetatable(io.stderr)
assert(mt == mt2, "io.stdin and io.stdout should share same metatable")
assert(mt == mt3, "io.stdin and io.stderr should share same metatable")

-- Test 3: Opened files share the same metatable
local f = io.tmpfile()
local mt4 = getmetatable(f)
assert(mt == mt4, "io.tmpfile() should share same metatable as io.stdin")
f:close()

-- Test 4: The metatable has expected fields
assert(mt.__name == "FILE*", "__name should be 'FILE*', got " .. tostring(mt.__name))
assert(type(mt.__index) == "table", "__index should be a table")
assert(type(mt.__tostring) == "function", "__tostring should be a function")
assert(type(mt.__close) == "function", "__close should be a function")
assert(type(mt.__gc) == "function", "__gc should be a function")

-- Test 5: __index table contains file methods
local idx = mt.__index
assert(type(idx.read) == "function", "__index should contain 'read'")
assert(type(idx.write) == "function", "__index should contain 'write'")
assert(type(idx.close) == "function", "__index should contain 'close'")
assert(type(idx.seek) == "function", "__index should contain 'seek'")
assert(type(idx.lines) == "function", "__index should contain 'lines'")
assert(type(idx.flush) == "function", "__index should contain 'flush'")
assert(type(idx.setvbuf) == "function", "__index should contain 'setvbuf'")

-- Test 6: debug.getmetatable also works
local dmt = debug.getmetatable(io.stdin)
assert(dmt == mt, "debug.getmetatable should return same metatable")

print("PASS: file handle metatable tests")
