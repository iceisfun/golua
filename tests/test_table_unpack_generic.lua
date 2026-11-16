-- table.unpack accepts any value, not just tables (uses metamethods)
-- Bug: GoLua required table type; Lua 5.4 uses generic len/index

-- String: has length, but s[i] returns nil
local a, b, c, d, e = table.unpack("hello")
assert(a == nil and b == nil and c == nil and d == nil and e == nil,
    "unpack string should return nils")

-- Object with __len and __index
local mt = {
    __len = function() return 3 end,
    __index = function(_, k) return k * 10 end
}
local obj = setmetatable({}, mt)
local x, y, z = table.unpack(obj)
assert(x == 10 and y == 20 and z == 30,
    "unpack with __len/__index: " .. tostring(x) .. "," .. tostring(y) .. "," .. tostring(z))

-- Lua 5.5 rejects values that are neither tables nor indexable with a length
local ok1, err1 = pcall(table.unpack, 42)
assert(not ok1 and err1:find("table expected, got number", 1, true), "unpack(42): " .. tostring(err1))

local ok2, err2 = pcall(table.unpack, true)
assert(not ok2 and err2:find("table expected, got boolean", 1, true), "unpack(true): " .. tostring(err2))

local ok3, err3 = pcall(table.unpack, nil)
assert(not ok3 and err3:find("table expected, got nil", 1, true), "unpack(nil): " .. tostring(err3))

-- Normal table still works
local a, b, c = table.unpack({10, 20, 30})
assert(a == 10 and b == 20 and c == 30)

