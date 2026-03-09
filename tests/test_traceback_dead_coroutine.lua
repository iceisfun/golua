-- Test: debug.traceback on dead/errored coroutine shows stack at error point
local co = coroutine.create(function()
    local function inner() error("fail") end
    inner()
end)
local ok, err = coroutine.resume(co)
assert(not ok)

local tb = debug.traceback(co, "DEAD", 0)
assert(type(tb) == "string", "expected string")
assert(tb:find("DEAD"), "expected DEAD in traceback")
-- Should show the stack at the error point, not an empty traceback
assert(tb:find("inner"), "expected 'inner' in traceback, got: " .. tb)

print("OK")
