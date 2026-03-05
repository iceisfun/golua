-- 1. Test Upvalue Capture vs Reused Call Buffers
local function make_adder(x)
    return function(y) return x + y end
end

local add5 = make_adder(5)
local add10 = make_adder(10)

print(add5(2))
--> =7
print(add10(2))
--> =12

-- 2. Test retBuf safety during __close (The "Clobber" Test)
-- This ensures that when 'return' triggers a close, the return
-- values are preserved even if the close logic calls more functions.

local clobber_triggered = false
local function clobber_buffer()
    -- This function call uses the VM's retBuf
    local internal = function() return "clobber" end
    internal()
    clobber_triggered = true
end

local function test_close_safety()
    -- Create a to-be-closed variable
    local _ <close> = setmetatable({}, {
        __close = function()
            clobber_buffer()
        end
    })

    return "original_value", 42
end

local v1, v2 = test_close_safety()
print(v1)
--> =original_value
print(v2)
--> =42
print(clobber_triggered)
--> =true

-- 3. Test Call Buffer Overflow (More than 8 args)
-- This triggers the fallback from [8]Value stack array to heap slice
local function sum_many(a, b, c, d, e, f, g, h, i, j)
    return a + b + c + d + e + f + g + h + i + j
end

print(sum_many(1, 1, 1, 1, 1, 1, 1, 1, 1, 1))
--> =10

-- 4. Nested Coroutine Return Safety
-- Ensures retBuf is handled correctly across coroutine boundaries
local co = coroutine.create(function()
    coroutine.yield("yield_val")
    return "return_val"
end)

local _, res1 = coroutine.resume(co)
print(res1)
--> =yield_val
local _, res2 = coroutine.resume(co)
print(res2)
--> =return_val
