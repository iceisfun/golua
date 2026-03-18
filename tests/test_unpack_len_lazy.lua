-- table.unpack should NOT call __len when j is explicitly provided.
local t = setmetatable({10, 20, 30}, {
  __len = function() error("__len should not be called") end,
})

-- Explicit j=2: __len must not fire.
local a, b = table.unpack(t, 1, 2)
assert(a == 10, "expected 10, got " .. tostring(a))
assert(b == 20, "expected 20, got " .. tostring(b))

-- Explicit j=3
local x, y, z = table.unpack(t, 1, 3)
assert(x == 10 and y == 20 and z == 30)

-- Without j: __len IS called (should error).
local ok, err = pcall(table.unpack, t, 1)
assert(not ok, "expected error when j is omitted")
assert(tostring(err):find("__len should not be called"), "wrong error: " .. tostring(err))

print("OK")
