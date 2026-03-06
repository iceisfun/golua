-- Test that stack overflow errors include source location prefix.

local ok, err = pcall(function()
  local function f() return 1 + f() end
  f()
end)
assert(not ok)
-- Error should contain "stack overflow" with a source:line: prefix
assert(err:match(":%d+: stack overflow"), "expected location prefix, got: " .. tostring(err))

-- Also test simple recursion
local function g() g() end
local ok2, err2 = pcall(g)
assert(not ok2)
assert(err2:match(":%d+: stack overflow"), "expected location prefix, got: " .. tostring(err2))

print("PASS")
