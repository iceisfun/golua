-- Test: coroutine.resume must preserve non-string error objects
-- Bug: runCoroutine wraps panics with fmt.Errorf, and coResume
-- calls err.Error(), both of which stringify the original Lua value.

-- 1. Table error object must survive resume
local err_obj = {code = 404, msg = "not found"}
local co = coroutine.create(function()
    error(err_obj, 0)
end)
local ok, val = coroutine.resume(co)
assert(not ok, "resume should return false on error")
assert(type(val) == "table",
    "error object should be a table, got: " .. type(val))
assert(val == err_obj, "error object identity should be preserved")
assert(val.code == 404, "error object fields should be accessible")

-- 2. Error after yield preserves object
local co2 = coroutine.create(function()
    coroutine.yield("before")
    error({kind = "late_error"}, 0)
end)
local ok1, v1 = coroutine.resume(co2)
assert(ok1 and v1 == "before")
local ok2, v2 = coroutine.resume(co2)
assert(not ok2, "second resume should fail")
assert(type(v2) == "table",
    "late error object should be a table, got: " .. type(v2))
assert(v2.kind == "late_error", "late error fields should be accessible")

-- 3. String errors still work
local co3 = coroutine.create(function()
    error("simple string", 0)
end)
local ok3, v3 = coroutine.resume(co3)
assert(not ok3)
assert(type(v3) == "string" and v3 == "simple string",
    "string error should be preserved, got: " .. tostring(v3))

-- 4. Number error object
local co4 = coroutine.create(function()
    error(42, 0)
end)
local ok4, v4 = coroutine.resume(co4)
assert(not ok4)
assert(v4 == 42, "number error should be preserved, got: " .. tostring(v4))

-- 5. Boolean error object
local co5 = coroutine.create(function()
    error(false, 0)
end)
local ok5, v5 = coroutine.resume(co5)
assert(not ok5)
assert(v5 == false, "boolean error should be preserved")
