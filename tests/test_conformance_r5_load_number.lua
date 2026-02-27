-- BUG: load(number) panics instead of coercing to string
-- In Lua 5.4, load() accepts a number as the first argument by
-- converting it to a string. GoLua panics with "bad argument #1
-- to 'load' (function expected, got number)".

-- load(integer) should convert to string and compile it
local f, err = load(123)
-- "123" is not a valid Lua chunk, so f should be nil with an error
assert(f == nil, "load(123) should return nil (not valid Lua)")
assert(type(err) == "string", "load(123) should return error string, got: " .. type(err))

-- Verify it doesn't panic
local ok, result = pcall(load, 42)
assert(ok == true, "load(number) should not panic, got: " .. tostring(result))

-- load(0) — also not valid Lua
local f2, err2 = load(0)
assert(f2 == nil, "load(0) should return nil (not valid Lua)")
assert(type(err2) == "string", "load(0) should return error string")

-- load(float) should also coerce to string
local f3, err3 = load(1.5)
assert(f3 == nil, "load(1.5) should return nil (not valid Lua)")
assert(type(err3) == "string", "load(1.5) should return error string")

local ok4, result4 = pcall(load, 3.14)
assert(ok4 == true, "load(float) should not panic, got: " .. tostring(result4))

-- load(boolean) should error (not coercible)
local ok5, err5 = pcall(load, true)
assert(not ok5, "load(true) should error")
assert(type(err5) == "string", "load(true) error should be string")
assert(err5:find("function expected"), "load(true) should say 'function expected', got: " .. err5)

-- load(nil) should error
local ok6, err6 = pcall(load, nil)
assert(not ok6, "load(nil) should error")
assert(type(err6) == "string", "load(nil) error should be string")

-- load(table) should error
local ok7, err7 = pcall(load, {})
assert(not ok7, "load({}) should error")
assert(type(err7) == "string", "load({}) error should be string")
assert(err7:find("function expected"), "load({}) should say 'function expected', got: " .. err7)
