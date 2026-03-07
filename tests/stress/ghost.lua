-- Stress test: upvalue survives coroutine collection

local function probe()
    local co = coroutine.create(function()
        local secret = "alive"
        return function() return secret end
    end)

    local _, get_secret = coroutine.resume(co)
    return get_secret
end

local leaked_closure = probe()
collectgarbage("collect")

local ok, result = pcall(leaked_closure)
assert(ok and result == "alive", "upvalue lost or corrupted after coroutine GC")
