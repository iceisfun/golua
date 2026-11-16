-- Test: mixing reads and writes on one r+ file without an intervening seek
-- acts at the logical position (glibc behavior), not at the read-ahead or
-- before unflushed writes.
local name = os.tmpname()

local f = io.open(name, "w") f:write("abcd") f:close()
f = io.open(name, "r+")
f:write("Y")
assert(f:read(1) == "b")
f:seek("cur", 0)
f:write("Q")
f:close()
f = io.open(name)
assert(f:read("a") == "YbQd")
f:close()

f = io.open(name, "w") f:write("abcd") f:close()
f = io.open(name, "r+")
assert(f:read(2) == "ab")
f:write("__")
f:seek("set")
assert(f:read("a") == "ab__")
f:close()

os.remove(name)
print("PASS")
