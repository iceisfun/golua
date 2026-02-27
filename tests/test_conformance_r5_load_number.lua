-- BUG: load(number) panics instead of coercing to string
-- In Lua 5.4, load() accepts a number as the first argument by
-- converting it to a string. GoLua panics with "bad argument #1
-- to 'load' (function expected, got number)".

-- load(number) should convert number to string and compile it
local f, err = load(123)
-- "123" is not a valid Lua chunk, so f should be nil with an error
assert(f == nil, "load(123) should return nil (not valid Lua)")
assert(type(err) == "string", "load(123) should return error string, got: " .. type(err))

-- Verify it doesn't panic
local ok, result = pcall(load, 42)
assert(ok == true, "load(number) should not panic, got: " .. tostring(result))

-- load(number) that IS valid Lua when converted to string
-- "return 1" can't be a number, but we can test "0" which is not a valid chunk
local f2, err2 = load(0)
assert(f2 == nil, "load(0) should return nil (not valid Lua)")
assert(type(err2) == "string", "load(0) should return error string")
