-- xpcall coroutine/yield boundary behavior.
-- Lua 5.4 semantics:
-- - the protected function may yield
-- - the message handler runs in a non-yieldable C boundary

do
    local co = coroutine.create(function()
        return xpcall(function()
            coroutine.yield("Y")
            return 99
        end, function(e)
            return "H:" .. tostring(e)
        end)
    end)

    print(coroutine.resume(co))
    --> =true	Y
    print(coroutine.resume(co))
    --> =true	true	99
end

do
    local co = coroutine.create(function()
        return xpcall(function()
            error("boom")
        end, function(e)
            return coroutine.yield("H", e)
        end)
    end)

    print(coroutine.resume(co))
    --> =true	false	error in error handling
end
