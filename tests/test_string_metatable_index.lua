-- Bug: String metatable is missing __index = string.
-- In Lua 5.4, getmetatable("").__index is the string table itself.
-- golua puts methods directly on the metatable but __index is nil.

local mt = getmetatable("")
assert(mt ~= nil, "string metatable should exist")

-- __index should be the string table
assert(mt.__index ~= nil, "string metatable should have __index")
assert(type(mt.__index) == "table", "__index should be a table, got " .. type(tostring(mt.__index)))
assert(mt.__index == string, "__index should be the string table")

-- Verify methods are accessible via the string table
assert(type(string.upper) == "function", "string.upper should be a function")
assert(type(string.lower) == "function", "string.lower should be a function")

-- Method calls should still work
assert(("hello"):upper() == "HELLO")

print("PASS")
