-- Test that pcall(xpcall, non_callable, handler) preserves the false status

-- Case 1: nil first arg, handler returns nothing
local ok, status, msg = pcall(xpcall, nil, function() end)
assert(ok == true, "pcall should succeed, got: " .. tostring(ok))
assert(status == false, "xpcall status should be false, got: " .. tostring(status))

-- Case 2: nil first arg, handler returns a value
local ok2, status2, msg2 = pcall(xpcall, nil, function() return "v" end)
assert(ok2 == true, "pcall should succeed, got: " .. tostring(ok2))
assert(status2 == false, "xpcall status should be false, got: " .. tostring(status2))
assert(msg2 == "v", "handler result should be 'v', got: " .. tostring(msg2))

-- Case 3: true first arg, handler returns nothing (should be exactly 2 results)
local ok3, status3 = pcall(xpcall, true, function() end)
assert(ok3 == true, "pcall should succeed")
assert(status3 == false, "xpcall status should be false, got: " .. tostring(status3))

-- Case 4: number first arg
local ok4, status4 = pcall(xpcall, 42, function() end)
assert(ok4 == true)
assert(status4 == false, "xpcall status should be false for number arg, got: " .. tostring(status4))

print("PASS")
