local function f() f() end
local ok, err = pcall(f)
assert(string.find(err, "stack overflow"), "expected 'stack overflow', got: " .. tostring(err))
-- Pure Lua recursion should NOT say "C stack overflow"
assert(not string.find(err, "C stack overflow"), "pure Lua recursion should say 'stack overflow', not 'C stack overflow', got: " .. tostring(err))
print("OK")
