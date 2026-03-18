-- Test that # on thread produces an error (not 0)
local co = coroutine.create(function() end)
local ok, err = pcall(function() return #co end)
assert(not ok, "expected error for #thread")
assert(err:find("attempt to get length of a thread value"),
  "wrong error message: " .. tostring(err))
print("OK")
