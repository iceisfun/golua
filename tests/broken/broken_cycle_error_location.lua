-- BUG: __index/__newindex cycle errors lack source location prefix
-- Reference Lua 5.4: "<file>:<line>: '__index' chain too long; possible loop"
-- GoLua:             "'__index' chain too long; possible loop"

-- __index cycle
local t1, t2 = {}, {}
setmetatable(t1, {__index = t2})
setmetatable(t2, {__index = t1})
local ok, err = pcall(function() return t1.x end)
assert(not ok)
-- Error message should contain a colon (file:line: prefix)
assert(type(err) == "string", "error should be a string")
assert(err:find(":%d+:"), "error should have source location, got: " .. err)
assert(err:find("'__index' chain too long"), "got: " .. err)

-- __newindex cycle
local a, b = {}, {}
setmetatable(a, {__newindex = b})
setmetatable(b, {__newindex = a})
local ok2, err2 = pcall(function() a.x = 1 end)
assert(not ok2)
assert(err2:find(":%d+:"), "error should have source location, got: " .. err2)
assert(err2:find("'__newindex' chain too long"), "got: " .. err2)

print("PASSED")
