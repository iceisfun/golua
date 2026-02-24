-- Coroutine hardening tests: edge cases for error handling, wrapping,
-- nested resumes, and value passing.

-- 1. wrap() propagates table errors through pcall
local gen_err = coroutine.wrap(function()
    error({code = "WRAP_ERR"}, 0)
end)
local ok, err = pcall(gen_err)
assert(not ok, "wrap error should propagate")
assert(type(err) == "table", "wrap should preserve table error, got: " .. type(err))
assert(err.code == "WRAP_ERR", "wrap error fields should be accessible")

-- 2. Error object preserved across yield boundary
local co_yield_err = coroutine.create(function()
    coroutine.yield("checkpoint")
    error({stage = "post_yield"}, 0)
end)
local ok1, v1 = coroutine.resume(co_yield_err)
assert(ok1 and v1 == "checkpoint")
local ok2, v2 = coroutine.resume(co_yield_err)
assert(not ok2)
assert(type(v2) == "table" and v2.stage == "post_yield",
    "post-yield error should preserve table")

-- 3. Nested coroutine error propagation
-- Inner coroutine errors, outer coroutine catches via resume
local inner = coroutine.create(function()
    error({level = "inner"}, 0)
end)
local outer = coroutine.create(function()
    local ok, err = coroutine.resume(inner)
    -- ok should be false, err should be the table
    return ok, err
end)
local s, rok, rerr = coroutine.resume(outer)
assert(s == true, "outer resume should succeed")
assert(rok == false, "inner resume should have failed")
assert(type(rerr) == "table" and rerr.level == "inner",
    "nested error object should be preserved")

-- 4. Multiple values through yield
local co_multi = coroutine.create(function()
    local a, b, c = coroutine.yield(1, 2, 3)
    return a .. b .. c
end)
local m1, a1, b1, c1 = coroutine.resume(co_multi)
assert(m1 and a1 == 1 and b1 == 2 and c1 == 3,
    "multi-value yield failed")
local m2, r = coroutine.resume(co_multi, "x", "y", "z")
assert(m2 and r == "xyz", "multi-value resume failed, got: " .. tostring(r))

-- 5. Zero values through yield
local co_zero = coroutine.create(function()
    coroutine.yield()
    return "done"
end)
local z1 = coroutine.resume(co_zero)
assert(z1 == true, "zero-value yield should return true")
local z2, zr = coroutine.resume(co_zero)
assert(z2 and zr == "done")

-- 6. Coroutine as producer with wrap
local function range(n)
    return coroutine.wrap(function()
        for i = 1, n do
            coroutine.yield(i)
        end
    end)
end
local sum = 0
for v in range(10) do
    sum = sum + v
end
assert(sum == 55, "wrap producer sum should be 55, got: " .. sum)

-- 7. Resume dead coroutine returns error string
local co_dead = coroutine.create(function() return 1 end)
coroutine.resume(co_dead)
assert(coroutine.status(co_dead) == "dead")
local dok, derr = coroutine.resume(co_dead)
assert(not dok, "resume dead should fail")
assert(type(derr) == "string", "resume dead error should be string")

-- 8. Coroutine that returns nil explicitly
local co_nil = coroutine.create(function()
    return nil
end)
local nok, nval = coroutine.resume(co_nil)
assert(nok == true, "resume should succeed")
assert(nval == nil, "should return nil")

-- 9. Deep yield chain (coroutine A yields, B yields, cascading)
local function make_yielder(depth)
    return coroutine.create(function(val)
        if depth > 0 then
            local inner = make_yielder(depth - 1)
            local ok, result = coroutine.resume(inner, val + 1)
            coroutine.yield(result)
        else
            coroutine.yield(val)
        end
    end)
end
local deep = make_yielder(10)
local dok2, dval = coroutine.resume(deep, 0)
assert(dok2 and dval == 10,
    "deep yield chain should return 10, got: " .. tostring(dval))

-- 10. Error in wrap doesn't corrupt subsequent pcall
local bad_wrap = coroutine.wrap(function()
    error("wrap_fail", 0)
end)
local wok1, werr1 = pcall(bad_wrap)
assert(not wok1)
-- Now a normal pcall should still work fine
local wok2, wval2 = pcall(function() return 42 end)
assert(wok2 and wval2 == 42, "pcall after wrap error should work")
