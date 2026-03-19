-- Suspended coroutine C frames should reject out-of-range debug.setlocal
-- slots instead of accepting them.

local co = coroutine.create(function()
    coroutine.yield()
end)

coroutine.resume(co)
print(debug.setlocal(co, 0, 2, 123))
--> =nil
