-- Test: stdlib functions must error when called with no arguments
-- Lua 5.4 requires these functions to validate that at least one argument is provided.
-- GoLua silently returns nil/default values instead of erroring.

-- type() requires exactly one argument
local ok, err = pcall(type)
assert(not ok, "type() should error with no args")
assert(string.find(tostring(err), "value expected"), "type() error message: " .. tostring(err))

-- tostring() requires exactly one argument
local ok2, err2 = pcall(tostring)
assert(not ok2, "tostring() should error with no args")
assert(string.find(tostring(err2), "value expected"), "tostring() error message: " .. tostring(err2))

-- tonumber() requires at least one argument
local ok3, err3 = pcall(tonumber)
assert(not ok3, "tonumber() should error with no args")
assert(string.find(tostring(err3), "value expected"), "tonumber() error message: " .. tostring(err3))

-- rawequal() requires two arguments
local ok4, err4 = pcall(rawequal)
assert(not ok4, "rawequal() should error with no args")
assert(string.find(tostring(err4), "value expected"), "rawequal() error message: " .. tostring(err4))

-- getmetatable() requires one argument
local ok5, err5 = pcall(getmetatable)
assert(not ok5, "getmetatable() should error with no args")
assert(string.find(tostring(err5), "value expected"), "getmetatable() error message: " .. tostring(err5))

-- math.type() requires one argument
local ok6, err6 = pcall(math.type)
assert(not ok6, "math.type() should error with no args")
assert(string.find(tostring(err6), "value expected"), "math.type() error message: " .. tostring(err6))

-- math.tointeger() requires one argument
local ok7, err7 = pcall(math.tointeger)
assert(not ok7, "math.tointeger() should error with no args")
assert(string.find(tostring(err7), "value expected"), "math.tointeger() error message: " .. tostring(err7))
