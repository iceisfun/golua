local function probe()
    local co = coroutine.create(function()
        local secret = "alive"
        -- This closure captures 'secret' from this coroutine's stack
        return function() return secret end
    end)

    local _, get_secret = coroutine.resume(co)
    -- 'co' is now suspended, 'secret' is an OPEN upvalue on co's stack
    return get_secret
end

local leaked_closure = probe()

-- Force Go/Lua to collect the coroutine 'co'
collectgarbage("collect")

-- Call the closure. If the coroutine stack was freed/corrupted, this fails.
local ok, result = pcall(leaked_closure)
if ok and result == "alive" then
    print("✓ Success: Upvalue survived coroutine collection")
else
    print("!! Failure: Upvalue lost or corrupted after coroutine GC")
end