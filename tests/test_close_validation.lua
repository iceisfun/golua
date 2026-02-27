-- Bug 1: <close> variables don't validate that values are closeable.
-- Non-nil values without __close should error at assignment time.
-- Bug 2: break/goto don't call __close on to-be-closed variables.

-- Test 1: assigning integer to <close> should error
local ok1, err1 = pcall(function()
  local x <close> = 42
end)
assert(not ok1, "integer assigned to <close> should error")
assert(err1:find("non%-closable") or err1:find("closeable") or err1:find("close"),
  "error should mention non-closable: " .. tostring(err1))

-- Test 2: assigning string to <close> should error
local ok2, err2 = pcall(function()
  local x <close> = "hello"
end)
assert(not ok2, "string assigned to <close> should error")

-- Test 3: assigning table without __close should error
local ok3, err3 = pcall(function()
  local x <close> = {}
end)
assert(not ok3, "table without __close assigned to <close> should error")

-- Test 4: nil is OK for <close>
local ok4 = pcall(function()
  local x <close> = nil
end)
assert(ok4, "nil should be valid for <close>")

-- Test 5: false is OK for <close> (Lua 5.4.4+)
local ok5 = pcall(function()
  local x <close> = false
end)
assert(ok5, "false should be valid for <close>")

-- Test 6: table with __close is OK
local ok6 = pcall(function()
  local x <close> = setmetatable({}, {__close = function() end})
end)
assert(ok6, "table with __close should be valid for <close>")

-- Test 7: break should trigger __close
local closed_on_break = false
for i = 1, 10 do
  local f <close> = setmetatable({}, {
    __close = function() closed_on_break = true end
  })
  break
end
assert(closed_on_break, "break should call __close on to-be-closed variables")

-- Test 8: normal scope exit calls __close (baseline)
local closed_normal = false
do
  local f <close> = setmetatable({}, {
    __close = function() closed_normal = true end
  })
end
assert(closed_normal, "normal scope exit should call __close")

print("PASS")
