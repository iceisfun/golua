-- debug.getinfo on coroutine.wrap function should report nups=1
-- The wrapped coroutine thread is an upvalue of the C wrapper function

local co = coroutine.wrap(function() end)
local info = debug.getinfo(co, "u")
assert(info.nups == 1, "expected nups=1, got " .. tostring(info.nups))
print("OK")
