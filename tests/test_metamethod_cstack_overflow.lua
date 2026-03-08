-- Test: metamethod recursion should produce "C stack overflow"
-- Bug: golua says "stack overflow" for metamethod recursion, but lua5.4 says "C stack overflow"

-- __tostring that calls tostring(self)
local t1 = setmetatable({}, {__tostring = function(self) return tostring(self) end})
local ok1, err1 = pcall(tostring, t1)
assert(not ok1)
assert(string.find(err1, "C stack overflow"), "expected 'C stack overflow' for __tostring recursion, got: " .. tostring(err1))

-- __index function that accesses self.x
local t2 = setmetatable({}, {__index = function(self, k) return self.x end})
local ok2, err2 = pcall(function() return t2.x end)
assert(not ok2)
assert(string.find(err2, "C stack overflow"), "expected 'C stack overflow' for __index function recursion, got: " .. tostring(err2))

-- __add that does a + b
local t3 = setmetatable({}, {__add = function(a, b) return a + b end})
local ok3, err3 = pcall(function() return t3 + t3 end)
assert(not ok3)
assert(string.find(err3, "C stack overflow"), "expected 'C stack overflow' for __add recursion, got: " .. tostring(err3))

-- Pure Lua recursion should still say "stack overflow" (not "C stack overflow")
local function f() f() end
local ok4, err4 = pcall(f)
assert(not ok4)
assert(string.find(err4, "stack overflow"), "expected 'stack overflow' for pure Lua recursion, got: " .. tostring(err4))
assert(not string.find(err4, "C stack overflow"), "pure Lua recursion should NOT say 'C stack overflow', got: " .. tostring(err4))

print("OK")
