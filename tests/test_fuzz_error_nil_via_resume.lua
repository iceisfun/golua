-- Lua 5.5: error(nil) (and other non-standard error objects) is replaced by
-- the string "<no error object>" when returned from a protected call. This
-- applies to the coroutine.resume path too, not just pcall/xpcall.

local co = coroutine.create(function() error(nil) end)
local ok, err = coroutine.resume(co)
assert(ok == false, "resume should return false")
assert(err == "<no error object>",
  "expected '<no error object>', got: " .. tostring(err) .. " (" .. type(err) .. ")")

print("PASSED")
