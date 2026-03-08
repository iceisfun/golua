-- Test that file:seek("cur", 0) returns correct position after buffered reads.
-- After seek("set", 5) + read(3), the logical position should be 8,
-- not the OS file descriptor position which may be ahead due to bufio read-ahead.

local tmpname = os.tmpname()
local f = io.open(tmpname, "w")
f:write("0123456789")
f:close()

f = io.open(tmpname, "r")
f:seek("set", 5)
f:read(3)
local pos = f:seek("cur", 0)
assert(pos == 8, "expected 8, got " .. tostring(pos))
f:close()

-- Also test that seek("cur") with a non-zero offset works correctly
f = io.open(tmpname, "r")
f:seek("set", 2)
f:read(3) -- read 3 bytes, logical pos = 5
local pos2 = f:seek("cur", 2) -- should be 7
assert(pos2 == 7, "expected 7, got " .. tostring(pos2))
f:close()

os.remove(tmpname)
print("OK")
