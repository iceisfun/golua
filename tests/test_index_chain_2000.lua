-- Test: __index chain of 2000 redirects should succeed (matching lua5.4 MAXTAGLOOP=2000)
-- Bug: golua only allows 1999 redirects due to off-by-one in loop condition

local N = 2000
local tabs = {}
for i = 0, N do
  tabs[i] = {}
end
-- Put the value in the last table
tabs[N].x = "found"
-- Set __index chain
for i = 0, N-1 do
  setmetatable(tabs[i], {__index = tabs[i+1]})
end

-- 2000 redirects should succeed
local ok, result = pcall(function() return tabs[0].x end)
assert(ok, "2000 __index redirects should succeed, got error: " .. tostring(result))
assert(result == "found", "expected 'found', got: " .. tostring(result))

-- 2001 redirects should fail
local tabs2 = {}
for i = 0, N+1 do
  tabs2[i] = {}
end
tabs2[N+1].x = "found"
for i = 0, N do
  setmetatable(tabs2[i], {__index = tabs2[i+1]})
end
local ok2, err2 = pcall(function() return tabs2[0].x end)
assert(not ok2, "2001 __index redirects should fail")
assert(string.find(err2, "chain too long"), "expected 'chain too long' error, got: " .. tostring(err2))

print("OK")
