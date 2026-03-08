-- Bug: debug.upvaluejoin error message includes index number
-- Lua 5.4 says "invalid upvalue index" (no number)
-- GoLua says "invalid upvalue index 1" (with number)

local function f() return 1 end  -- no upvalues (except _ENV which is stripped for top-level?)

-- Try joining with invalid index
local ok, err = pcall(debug.upvaluejoin, f, 1, f, 1)
assert(not ok, "should fail")
-- The error message should NOT include the index number
assert(err:find("invalid upvalue index%)$") ~= nil,
  "error should end with 'invalid upvalue index)' but got: " .. err)

print("PASSED")
