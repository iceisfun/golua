-- Test: stripped function error() should not add spurious source prefix
local f = function() error("test") end
local s = string.dump(f, true) -- strip debug info
local g = load(s, "=test")
local ok, err = pcall(g)
assert(err == "test", "expected 'test', got: " .. tostring(err))

-- Also test with a non-stripped function for comparison
local f2 = function() error("hello") end
local s2 = string.dump(f2) -- don't strip
local g2 = load(s2, "=test2")
local ok2, err2 = pcall(g2)
-- Non-stripped should have source prefix
assert(string.find(err2, "hello"), "expected 'hello' in error, got: " .. tostring(err2))

print("OK")
