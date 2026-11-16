-- Lua 5.5 table.unpack always takes the length first (ltablib aux_getn), so
-- __len runs even when j is explicitly provided, and its errors propagate.
local calls = 0
local t = setmetatable({10, 20, 30}, {
  __len = function() calls = calls + 1; return 3 end,
})

local a, b = table.unpack(t, 1, 2)
assert(a == 10, "expected 10, got " .. tostring(a))
assert(b == 20, "expected 20, got " .. tostring(b))
assert(calls == 1, "__len should run once, ran " .. calls)

local x, y, z = table.unpack(t, 1, 3)
assert(x == 10 and y == 20 and z == 30)
assert(calls == 2)

-- An erroring __len fails the call whether or not j is given.
local bad = setmetatable({10, 20, 30}, {
  __len = function() error("__len called", 0) end,
})
local ok, err = pcall(table.unpack, bad, 1, 2)
assert(not ok and err == "__len called", "wrong error: " .. tostring(err))
ok, err = pcall(table.unpack, bad, 1)
assert(not ok and err == "__len called", "wrong error: " .. tostring(err))

print("OK")
