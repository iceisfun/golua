-- Bug: __index/__newindex with non-table, non-function values are
-- silently ignored instead of chaining through the value's metatable.

-- Test 1: __index with a number should error (no metatable to chain)
local t1 = setmetatable({}, {__index = 42})
local ok1, err1 = pcall(function() return t1.x end)
assert(not ok1, "__index=42 should error when indexed, got " .. tostring(ok1))
assert(err1:find("index"), "error should mention indexing: " .. tostring(err1))

-- Test 2: __index with a string should chain through string's metatable
-- string has metatable with __index = string table, so string methods
-- should be accessible
local t2 = setmetatable({}, {__index = "hello"})
local upper = t2.upper
assert(type(upper) == "function",
  "__index='hello' should find string.upper via string metatable, got " .. type(tostring(upper)))

-- Test 3: __newindex with a number should error
local t3 = setmetatable({}, {__newindex = 42})
local ok3, err3 = pcall(function() t3.x = 1 end)
assert(not ok3, "__newindex=42 should error when assigned")

print("PASS")
