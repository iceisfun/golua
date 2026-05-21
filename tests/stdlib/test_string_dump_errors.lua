-- string.dump error handling tests.
-- Lua 5.5's str_dump uses a single luaL_argcheck: every invalid arg #1 —
-- wrong type, missing, or a C/native function — produces the same
-- "(Lua function expected)" message.

-- Error: non-function argument
local ok, err = pcall(string.dump, 42)
assert(not ok)
assert(err:find("Lua function expected"), "got: " .. tostring(err))

local ok2, err2 = pcall(string.dump, "hello")
assert(not ok2)
assert(err2:find("Lua function expected"), "got: " .. tostring(err2))

local ok3, err3 = pcall(string.dump, true)
assert(not ok3)
assert(err3:find("Lua function expected"), "got: " .. tostring(err3))

local ok4, err4 = pcall(string.dump, nil)
assert(not ok4)
assert(err4:find("Lua function expected"), "got: " .. tostring(err4))

local ok5, err5 = pcall(string.dump, {})
assert(not ok5)
assert(err5:find("Lua function expected"), "got: " .. tostring(err5))

-- Error: native (C) function — same message
local ok6, err6 = pcall(string.dump, print)
assert(not ok6)
assert(err6:find("Lua function expected"), "got: " .. tostring(err6))

local ok7, err7 = pcall(string.dump, type)
assert(not ok7)
assert(err7:find("Lua function expected"), "got: " .. tostring(err7))

-- Loading corrupted binary should fail gracefully
local ok8, err8 = load("\x1bLua\x54garbage")
assert(ok8 == nil, "corrupted binary should not load")
assert(type(err8) == "string", "should return error message")

-- Loading binary with wrong version
local ok9, err9 = load("\x1bLua\x53\0\x19\x93\r\n\x1a\n")
assert(ok9 == nil, "wrong version should not load")

-- Truncated binary
local ok10, err10 = load("\x1bLua")
assert(ok10 == nil, "truncated should not load")

