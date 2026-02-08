-- Test: __index cycle is caught by pcall
local t1 = {}
local t2 = {}
setmetatable(t1, {__index = t2})
setmetatable(t2, {__index = t1})

local ok, err = pcall(function() return t1.missing end)
assert(not ok, "expected __index cycle to be caught")
assert(type(err) == "string", "expected error string, got " .. type(err))
assert(string.find(err, "__index"), "expected __index in error: " .. err)

-- Test: __newindex cycle is caught by pcall
local a = {}
local b = {}
setmetatable(a, {__newindex = b})
setmetatable(b, {__newindex = a})

local ok2, err2 = pcall(function() a.key = 42 end)
assert(not ok2, "expected __newindex cycle to be caught")
assert(type(err2) == "string", "expected error string, got " .. type(err2))
assert(string.find(err2, "__newindex"), "expected __newindex in error: " .. err2)

-- Test: valid __index chain works
local base = {value = 100}
local mid = {}
local top = {}
setmetatable(mid, {__index = base})
setmetatable(top, {__index = mid})

assert(top.value == 100, "expected 100 from chain, got " .. tostring(top.value))

-- Test: self-referencing __index cycle
local self_ref = {}
setmetatable(self_ref, {__index = self_ref})
local ok3, err3 = pcall(function() return self_ref.missing end)
assert(not ok3, "expected self-referencing __index cycle to be caught")
assert(string.find(err3, "__index"), "expected __index in error: " .. err3)
