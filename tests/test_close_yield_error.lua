-- coroutine.close with __close that yields should error
local co = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function() coroutine.yield() end})
    coroutine.yield(1)
end)
local ok, val = coroutine.resume(co)
assert(ok == true and val == 1, "first resume")
assert(coroutine.status(co) == "suspended", "suspended after yield")

local cok, cerr = coroutine.close(co)
assert(cok == false, "close should fail, got: " .. tostring(cok))
assert(type(cerr) == "string", "error should be string")
assert(cerr:find("yield across", 1, true), "expected yield error, got: " .. cerr)
