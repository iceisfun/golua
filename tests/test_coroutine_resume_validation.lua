-- Bug: coroutine.resume with non-thread arg returns error tuple
-- instead of raising it. pcall(coroutine.resume, {}) should return
-- false, "bad argument..." not true, false, "bad argument..."

-- Test: resume with table should raise error (not return it)
local ok, err = pcall(coroutine.resume, {})
assert(ok == false, "pcall should catch error from resume, got ok=" .. tostring(ok))
assert(type(err) == "string", "error should be string, got " .. type(err))
assert(err:find("thread expected"), "error should mention 'thread expected', got: " .. err)

-- Test: resume with number should raise error
local ok2, err2 = pcall(coroutine.resume, 42)
assert(ok2 == false, "pcall should catch error from resume with number")
assert(err2:find("thread expected"), "error should mention 'thread expected', got: " .. tostring(err2))

print("PASS")
