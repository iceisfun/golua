-- BROKEN: xpcall() not implemented
-- xpcall(f, msgh [, arg1, ...]) is a core Lua 5.4 function that calls f
-- in protected mode with a custom message handler.

-- Basic xpcall with error handler
local ok, msg = xpcall(function() error("boom") end, function(e) return "handled: " .. e end)
assert(not ok, "xpcall should return false on error")
assert(msg == "handled: input:5: boom" or msg:find("handled:"), "error handler should transform message")

-- xpcall with no error
local ok2, val2 = xpcall(function() return 42 end, function(e) return e end)
assert(ok2, "xpcall should return true on success")
assert(val2 == 42, "xpcall should return function result")

-- xpcall with extra arguments
local ok3, val3 = xpcall(function(a, b) return a + b end, function(e) return e end, 10, 20)
assert(ok3 and val3 == 30, "xpcall should pass extra arguments to function")
