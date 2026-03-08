-- Test: coroutine.resume on main thread says "non-suspended"

local main = coroutine.running()
local ok, err = coroutine.resume(main)
assert(ok == false, "should fail")
assert(err == "cannot resume non-suspended coroutine",
    "expected 'cannot resume non-suspended coroutine', got '" .. tostring(err) .. "'")

print("OK")
