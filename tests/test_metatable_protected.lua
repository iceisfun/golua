-- Test __metatable protection
local mt = {}
local t = setmetatable({}, mt)

assert(getmetatable(t) == mt)

mt.__metatable = "locked"
assert(getmetatable(t) == "locked")
