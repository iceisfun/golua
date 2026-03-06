-- BUG: Table index nil/NaN errors lack source location prefix
-- Reference Lua 5.4: "<file>:<line>: table index is nil"
-- GoLua:             "table index is nil"

-- nil key
local t = {}
local ok, err = pcall(function() t[nil] = 1 end)
assert(not ok)
assert(type(err) == "string", "error should be a string")
assert(err:find(":%d+:"), "nil key error should have source location, got: " .. err)
assert(err:find("table index is nil"), "got: " .. err)

-- NaN key
local ok2, err2 = pcall(function() t[0/0] = 1 end)
assert(not ok2)
assert(err2:find(":%d+:"), "NaN key error should have source location, got: " .. err2)
assert(err2:find("table index is NaN"), "got: " .. err2)

print("PASSED")
