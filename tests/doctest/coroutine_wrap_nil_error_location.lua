-- coroutine.wrap re-raises a coroutine error after prepending the caller
-- location (luaL_where(L,1)), exactly like luaB_auxwrap. Lua 5.5 converts a
-- nil error object to the string "<no error object>" at throw time, so by the
-- time wrap re-raises it is a string and gets the caller location prefixed.
-- Previously golua kept the nil value through the wrap re-raise and emitted the
-- bare "<no error object>" with no source:line: prefix.

local f = coroutine.wrap(function() error(nil) end)
local ok, err = pcall(function() return f() end)
print(ok, err)
--> ~^false\t.*:[0-9]+: <no error object>$

-- A non-nil string error still gets the caller location prepended.
local g = coroutine.wrap(function() error("boom") end)
local ok2, err2 = pcall(function() return g() end)
print(ok2, err2)
--> ~^false\t.*boom$

-- A table error object propagates unchanged (no location prefix).
local t = setmetatable({}, {__tostring = function() return "ERR_OBJ" end})
local h = coroutine.wrap(function() error(t) end)
local ok3, err3 = pcall(function() return h() end)
print(ok3, tostring(err3))
--> =false	ERR_OBJ
