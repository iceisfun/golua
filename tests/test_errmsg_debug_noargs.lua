-- Test that debug.getmetatable/setmetatable error with no args
local ok, err = pcall(debug.getmetatable)
assert(not ok, "expected error for debug.getmetatable()")
assert(err:find("value expected"), "wrong error: " .. tostring(err))

local ok2, err2 = pcall(debug.setmetatable)
assert(not ok2, "expected error for debug.setmetatable()")
assert(err2:find("nil or table expected"), "wrong error: " .. tostring(err2))
assert(err2:find("got no value"), "missing 'got no value': " .. tostring(err2))
print("OK")
