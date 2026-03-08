-- __name metafield should be used in error messages for all operation types,
-- not just arithmetic. Lua 5.4 uses __name for concat, comparison, call,
-- and bitwise error messages.

local mt = {__name = "MyObj"}
local t = setmetatable({}, mt)

-- concat error should say "MyObj" not "table"
local ok1, err1 = pcall(function() return t .. "x" end)
assert(err1:find("MyObj"), "concat error should use __name, got: " .. err1)

-- comparison error should say "MyObj"
local ok2, err2 = pcall(function() return t < t end)
assert(err2:find("MyObj"), "comparison error should use __name, got: " .. err2)

-- call error should say "MyObj"
local ok3, err3 = pcall(function() return t() end)
assert(err3:find("MyObj"), "call error should use __name, got: " .. err3)

-- bitwise error should say "MyObj"
local ok4, err4 = pcall(function() return t & 1 end)
assert(err4:find("MyObj"), "bitwise error should use __name, got: " .. err4)

-- le comparison error
local ok5, err5 = pcall(function() return t <= t end)
assert(err5:find("MyObj"), "le error should use __name, got: " .. err5)

-- Cross-type comparison should show both __names
local mt_a = {__name = "TypeA"}
local mt_b = {__name = "TypeB"}
local a = setmetatable({}, mt_a)
local b = setmetatable({}, mt_b)
local ok6, err6 = pcall(function() return a < b end)
assert(err6:find("TypeA"), "cross-type lt should mention TypeA, got: " .. err6)
assert(err6:find("TypeB"), "cross-type lt should mention TypeB, got: " .. err6)

print("OK")
