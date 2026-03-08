-- Test that TBC variables work with type-level metatables (__close on numbers, strings, booleans)

local results = {}

-- Test 1: number with type-level __close
debug.setmetatable(0, {__close = function(self) results[#results+1] = "num:" .. tostring(self) end})
do
    local x <close> = 42
end
debug.setmetatable(0, nil)
assert(results[1] == "num:42", "expected 'num:42', got: " .. tostring(results[1]))

-- Test 2: string with type-level __close
results = {}
debug.setmetatable("", {__close = function(self) results[#results+1] = "str:" .. self end})
do
    local x <close> = "hello"
end
debug.setmetatable("", nil)
assert(results[1] == "str:hello", "expected 'str:hello', got: " .. tostring(results[1]))

-- Test 3: boolean with type-level __close (true only; false is always OK/no-op)
results = {}
debug.setmetatable(true, {__close = function(self) results[#results+1] = "bool:" .. tostring(self) end})
do
    local x <close> = true
end
debug.setmetatable(true, nil)
assert(results[1] == "bool:true", "expected 'bool:true', got: " .. tostring(results[1]))

-- Test 4: multiple TBC with type metatables close in reverse order
results = {}
debug.setmetatable(0, {__close = function(self) results[#results+1] = tostring(self) end})
do
    local a <close> = 1
    local b <close> = 2
    local c <close> = 3
end
debug.setmetatable(0, nil)
assert(results[1] == "3", "expected '3', got: " .. tostring(results[1]))
assert(results[2] == "2", "expected '2', got: " .. tostring(results[2]))
assert(results[3] == "1", "expected '1', got: " .. tostring(results[3]))

-- Test 5: error value passed to __close
results = {}
debug.setmetatable(0, {__close = function(self, err)
    results[#results+1] = tostring(self) .. ":" .. tostring(err)
end})
local ok, err = pcall(function()
    local x <close> = 99
    error("boom")
end)
debug.setmetatable(0, nil)
assert(not ok)
-- The error value should be passed to __close
assert(string.find(results[1], "99:"), "expected error passed to __close, got: " .. tostring(results[1]))

print("PASS")
