-- Test io.lines() and f:lines() MAXARGLINE (250) limit

local fname = os.tmpname()

-- Create test file
local f = io.open(fname, "w")
f:write(string.rep("a", 300), "\n")
f:close()

-- 250 format args should work
local t = {}
for i = 1, 250 do t[i] = 1 end
t = {io.lines(fname, table.unpack(t))()}
assert(#t == 250, "expected 250 results, got " .. #t)
assert(t[1] == 'a', "first result should be 'a'")
assert(t[250] == 'a', "last result should be 'a'")
print("io.lines with 250 args: OK")

-- 251 format args should error
t = {}
for i = 1, 251 do t[i] = 1 end
local ok, err = pcall(io.lines, fname, table.unpack(t))
assert(not ok, "251 args should fail")
assert(string.find(err, "too many arguments", 1, true), "wrong error: " .. err)
print("io.lines with 251 args rejected: OK")

-- f:lines() should also enforce the limit
f = io.open(fname, "r")
t = {}
for i = 1, 251 do t[i] = 1 end
ok, err = pcall(f.lines, f, table.unpack(t))
assert(not ok, "f:lines with 251 args should fail")
assert(string.find(err, "too many arguments", 1, true), "wrong error: " .. err)
f:close()
print("f:lines with 251 args rejected: OK")

os.remove(fname)
print("PASS")
