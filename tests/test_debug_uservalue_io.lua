-- Test debug.getuservalue and debug.setuservalue on file handles (0 user value slots)

-- File handles should have 0 user value slots
-- getuservalue on file handle returns nil (1 value, no boolean)
local v1 = debug.getuservalue(io.stdin, 1)
assert(v1 == nil, "expected nil, got " .. tostring(v1))
assert(select("#", debug.getuservalue(io.stdin, 1)) == 1, "expected 1 return value")

-- setuservalue on file handle (0 slots) returns nil (not the userdata)
local r = debug.setuservalue(io.stdin, 10)
assert(r == nil, "expected nil for 0-slot userdata, got " .. tostring(r))
local r2 = debug.setuservalue(io.stdin, 10, 1)
assert(r2 == nil, "expected nil for n=1 on 0-slot userdata, got " .. tostring(r2))

print("OK")
