-- coroutine.close should not leak the suspended coroutine's old call name into
-- a __close handler's caller frame.

local co = coroutine.create(function()
    local obj = setmetatable({}, {
        __close = function()
            local i = debug.getinfo(2, "n")
            print(tostring(i and i.name), i and i.namewhat or "nil")
        end,
    })
    local x <close> = obj
    coroutine.yield("pause")
end)

coroutine.resume(co)
coroutine.close(co)
--> =nil	nil
