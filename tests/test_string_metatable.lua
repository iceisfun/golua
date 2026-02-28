-- getmetatable("str") should return the string metatable (or __metatable value)
-- In Lua 5.4, strings have a metatable with __index pointing to the string table
local mt = getmetatable("hello")
assert(mt ~= nil, "strings should have a metatable")
