-- Test io.open("") should fail, not open a directory
-- lua5.4 returns nil, ": No such file or directory"

local f, err = io.open("", "r")
assert(f == nil, "expected nil, got " .. tostring(f))
-- The error message should start with ":"  (empty filename followed by colon)
assert(type(err) == "string", "expected string error, got " .. type(err))
assert(err:sub(1,1) == ":", "expected error starting with ':', got: " .. err)

print("PASS")
