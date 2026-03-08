-- Test dofile and loadfile

-- loadfile loads a file without executing it
-- Write a temp helper file using io
local tmpname = os.tmpname()
local f = io.open(tmpname, "w")
f:write("return 42, 'hello'\n")
f:close()

-- loadfile returns a function
local fn, err = loadfile(tmpname)
assert(fn ~= nil, "loadfile should return a function, got nil: " .. tostring(err))
assert(type(fn) == "function", "loadfile should return a function, got " .. type(fn))

-- Calling the loaded function executes it
local a, b = fn()
assert(a == 42, "expected 42, got " .. tostring(a))
assert(b == "hello", "expected 'hello', got " .. tostring(b))

-- dofile loads and executes immediately
f = io.open(tmpname, "w")
f:write("return 99, 'world'\n")
f:close()

local c, d = dofile(tmpname)
assert(c == 99, "expected 99, got " .. tostring(c))
assert(d == "world", "expected 'world', got " .. tostring(d))

-- loadfile with mode parameter
f = io.open(tmpname, "w")
f:write("return 'text'\n")
f:close()

local fn2 = loadfile(tmpname, "t")
assert(fn2 ~= nil, "loadfile with mode 't' should work")
assert(fn2() == "text")

-- loadfile with env parameter
f = io.open(tmpname, "w")
f:write("return x\n")
f:close()

local env = {x = 123}
setmetatable(env, {__index = _G})
local fn3 = loadfile(tmpname, "t", env)
assert(fn3 ~= nil)
assert(fn3() == 123, "loadfile with env should use provided environment")

-- loadfile with non-existent file returns nil + error
local fn4, err4 = loadfile("nonexistent_file_xyz.lua")
assert(fn4 == nil, "loadfile of non-existent file should return nil")
assert(type(err4) == "string", "loadfile error should be a string")

-- Clean up
os.remove(tmpname)

print("PASS")
