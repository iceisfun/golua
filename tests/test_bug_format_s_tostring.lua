-- Bug: string.format("%s", obj) doesn't call __tostring metamethod.
-- Lua 5.4: %s calls luaL_tolstring which invokes __tostring if present.

-- Table with __tostring
local mt = setmetatable({}, {__tostring = function() return "META" end})
assert(tostring(mt) == "META", "tostring works")
local s = string.format("%s", mt)
assert(s == "META", "format %%s should call __tostring: got " .. s)

-- __tostring returning a number (Lua 5.4 converts to string)
local mt_num = setmetatable({}, {__tostring = function() return 42 end})
local s2 = string.format("%s", mt_num)
assert(s2 == "42", "format %%s with numeric __tostring: got " .. s2)

-- Multiple %s with mixed types
local mt2 = setmetatable({}, {__tostring = function() return "T2" end})
local s3 = string.format("%s and %s", mt2, "plain")
assert(s3 == "T2 and plain", "mixed format: got " .. s3)

-- Guard: %s with normal types should keep working
assert(string.format("%s", "hello") == "hello")
assert(string.format("%s", 42) == "42")
assert(string.format("%s", true) == "true")
assert(string.format("%s", nil) == "nil")
