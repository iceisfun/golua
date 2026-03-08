-- tostring() should use a protected call for __tostring invocation.
-- In Lua 5.4 (C implementation), luaL_tolstring uses luaL_callmeta which
-- does a protected call. If __tostring errors, tostring() catches the error
-- and returns the error message as a string (not propagating it).
--
-- This matters in xpcall handlers where tostring() on the error object
-- should not cause "error in error handling".

-- Test 1: __tostring that calls error()
local mt1 = {__tostring = function(self) error("ts_fail") end}
local obj1 = setmetatable({}, mt1)

local ok1, err1 = xpcall(function()
  error(obj1)
end, function(e)
  return "caught:" .. tostring(e)
end)
-- Lua 5.4: tostring catches __tostring error, handler succeeds
-- GoLua: tostring propagates error, handler fails -> "error in error handling"
assert(ok1 == false, "expected false from xpcall")
assert(type(err1) == "string")
assert(err1:find("caught:") ~= nil,
  "expected handler to succeed with 'caught:', got: " .. tostring(err1))
assert(err1:find("ts_fail") ~= nil,
  "expected error message to contain 'ts_fail', got: " .. tostring(err1))

-- Test 2: __tostring that returns wrong type
local mt2 = {__tostring = function(self) return {} end}
local obj2 = setmetatable({}, mt2)

local ok2, err2 = xpcall(function()
  error(obj2)
end, function(e)
  return "caught2:" .. tostring(e)
end)
assert(ok2 == false)
assert(type(err2) == "string")
assert(err2:find("caught2:") ~= nil,
  "expected handler to succeed with 'caught2:', got: " .. tostring(err2))

print("OK")
