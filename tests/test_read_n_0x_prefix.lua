-- Test read("n") on "0x" prefix followed by non-hex chars
-- Lua 5.4 returns nil for "0xZZZ" because "0x" alone is not a valid number

local tmpname = os.tmpname()

-- Test "0xZZZ" - should return nil
local f = io.open(tmpname, "w")
f:write("0xZZZ")
f:close()

f = io.open(tmpname, "r")
local val = f:read("n")
f:close()
assert(val == nil, "read('n') on '0xZZZ' should return nil, got: " .. tostring(val))

-- Test "0x" alone - should return nil
f = io.open(tmpname, "w")
f:write("0x")
f:close()

f = io.open(tmpname, "r")
val = f:read("n")
f:close()
assert(val == nil, "read('n') on '0x' should return nil, got: " .. tostring(val))

-- Test "0x1A" - should return 26
f = io.open(tmpname, "w")
f:write("0x1A")
f:close()

f = io.open(tmpname, "r")
val = f:read("n")
f:close()
assert(val == 0x1A, "read('n') on '0x1A' should return 26, got: " .. tostring(val))

os.remove(tmpname)
print("PASS")
