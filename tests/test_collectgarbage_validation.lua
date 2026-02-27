-- Bug: collectgarbage() with an invalid option silently returns 0
-- instead of raising an error.

-- Test 1: invalid option should error
local ok, err = pcall(collectgarbage, "invalid")
assert(not ok, "collectgarbage('invalid') should error, got " .. tostring(ok))
assert(err:find("invalid option"), "error should mention 'invalid option': " .. tostring(err))

-- Test 2: valid options should work (baseline)
assert(pcall(collectgarbage, "collect"))
assert(pcall(collectgarbage, "count"))
assert(pcall(collectgarbage, "stop"))
assert(pcall(collectgarbage, "restart"))

print("PASS")
