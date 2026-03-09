-- Test that string.gsub with unclosed captures and plain string replacement succeeds
-- Lua 5.4 only errors on unclosed captures when they're actually accessed

-- gsub with plain string replacement (no %N) should succeed
local r, n = string.gsub("hello", "(%w+", "x")
assert(r == "x", "expected 'x', got: " .. tostring(r))
assert(n == 1, "expected 1 replacement, got: " .. tostring(n))

-- gsub with %0 (whole match, not a capture) should also succeed
local r2, n2 = string.gsub("hello", "(%w+", "%0")
assert(r2 == "hello", "expected 'hello', got: " .. tostring(r2))
assert(n2 == 1)

-- gsub with %1 referencing the unclosed capture should fail
local ok, err = pcall(string.gsub, "hello", "(%w+", "%1")
assert(not ok, "should fail when %1 references unclosed capture")
assert(tostring(err):find("unfinished capture"), "expected unfinished capture error, got: " .. tostring(err))

-- find with unclosed capture should still fail
local ok2, err2 = pcall(string.find, "hello", "(%w+")
assert(not ok2, "find should fail with unclosed capture")

-- match with unclosed capture should still fail
local ok3, err3 = pcall(string.match, "hello", "(%w+")
assert(not ok3, "match should fail with unclosed capture")

-- gmatch with unclosed capture should still fail
local ok4, err4 = pcall(function()
    for w in string.gmatch("hello", "(%w+") do end
end)
assert(not ok4, "gmatch should fail with unclosed capture")

-- gsub with function replacement should fail (needs captures)
local ok5, err5 = pcall(string.gsub, "hello", "(%w+", function(c) return c end)
assert(not ok5, "gsub with function replacement should fail with unclosed capture")

-- gsub with table replacement should fail (needs captures)
local ok6, err6 = pcall(string.gsub, "hello", "(%w+", {hello="world"})
assert(not ok6, "gsub with table replacement should fail with unclosed capture")

print("OK")
