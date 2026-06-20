-- coroutine.close on a coroutine that died with error(nil) must report the
-- error object the same way resume does: a nil error object becomes the string
-- "<no error object>" (Lua 5.5). Previously close returned a raw nil.

do
    local co = coroutine.create(function() error(nil) end)
    print("resume", coroutine.resume(co))
    --> =resume	false	<no error object>
    print("close", coroutine.close(co))
    --> =close	false	<no error object>
end

-- The returned error object must be a string, not nil.
do
    local co = coroutine.create(function() error(nil) end)
    coroutine.resume(co)
    local ok, e = coroutine.close(co)
    print(ok, type(e))
    --> =false	string
end

-- A non-nil error object is still propagated unchanged.
do
    local co = coroutine.create(function() error("boom") end)
    coroutine.resume(co)
    local ok, e = coroutine.close(co)
    print(ok, e)
    --> ~^false\t.*boom$
end

-- A coroutine that errored with a table object propagates the table.
do
    local t = setmetatable({}, {__tostring = function() return "ERR_OBJ" end})
    local co = coroutine.create(function() error(t) end)
    coroutine.resume(co)
    local ok, e = coroutine.close(co)
    print(ok, tostring(e))
    --> =false	ERR_OBJ
end
