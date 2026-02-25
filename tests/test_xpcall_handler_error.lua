-- test_xpcall_handler_error: xpcall should report handler failures consistently

-- If message handler errors, Lua 5.4 returns a generic message.
local ok1, err1 = xpcall(function() error("boom") end, function(_) error("handler failed") end)
assert(ok1 == false, "xpcall should fail when handler errors")
assert(err1 == "error in error handling", "xpcall handler error should be generic")

-- Same behavior with non-string original errors.
local ok2, err2 = xpcall(function() error({k = 1}) end, function(_) error("handler failed", 0) end)
assert(ok2 == false, "xpcall should fail when handler errors (object error)")
assert(err2 == "error in error handling", "xpcall handler error should be generic for object errors")

-- If handler does not error and returns nil, xpcall should return false, nil.
local ok3, err3 = xpcall(function() error(nil) end, function(e) return e and e.k end)
assert(ok3 == false, "xpcall should fail for original nil error")
assert(err3 == nil, "handler return value should be propagated when no handler error")
