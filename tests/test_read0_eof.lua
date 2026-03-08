-- Test that file:read(0) returns nil at EOF (Lua 5.4 behavior)
-- read(0) is an EOF test: returns "" if more data available, nil at EOF

local tmpname = os.tmpname()

-- Write some data
local f = io.open(tmpname, "w")
f:write("hello")
f:close()

-- Read all data, then test read(0) at EOF
f = io.open(tmpname, "r")
f:read("a")  -- consume all data
local result = f:read(0)
assert(result == nil, "read(0) at EOF should return nil, got: " .. tostring(result))
f:close()

-- Test read(0) when NOT at EOF returns ""
f = io.open(tmpname, "r")
local result2 = f:read(0)
assert(result2 == "", "read(0) not at EOF should return empty string, got: " .. tostring(result2))
f:close()

-- Test read(0) on empty file returns nil
local tmpname2 = os.tmpname()
f = io.open(tmpname2, "w")
f:close()
f = io.open(tmpname2, "r")
local result3 = f:read(0)
assert(result3 == nil, "read(0) on empty file should return nil, got: " .. tostring(result3))
f:close()

os.remove(tmpname)
os.remove(tmpname2)

print("PASS")
