local function f() f() end
local ok, err = pcall(f)
assert(string.find(err, "C stack overflow"), "expected 'C stack overflow', got: " .. tostring(err))
print("OK")
