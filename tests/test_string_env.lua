-- String as _ENV should use string metatable __index
local f = load("return upper", nil, nil, "hello")
local ok, result = pcall(f)
-- 'upper' is a string method, so string.__index should find it
assert(ok, "string _ENV should work via __index, got: " .. tostring(result))
assert(type(result) == "function", "expected function, got: " .. type(result))

-- Non-existent key should return nil
local f2 = load("return x", nil, nil, "hello")
local ok2, result2 = pcall(f2)
assert(ok2, "string _ENV lookup of missing key should return nil, got: " .. tostring(result2))
assert(result2 == nil, "expected nil, got: " .. tostring(result2))

print("OK")
