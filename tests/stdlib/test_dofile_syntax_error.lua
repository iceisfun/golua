-- Bug: dofile/loadfile syntax errors should not include @ prefix in error message

-- Write a file with invalid syntax
local tmpfile = os.tmpname()
local f = io.open(tmpfile, "w")
f:write("local x = !!!\n")
f:close()

-- Test dofile
local ok, err = pcall(dofile, tmpfile)
assert(not ok, "dofile should fail on syntax error")
assert(not string.find(err, "@", 1, true), "dofile error should not contain @, got: " .. tostring(err))

-- Test loadfile
local fn, err2 = loadfile(tmpfile)
assert(fn == nil, "loadfile should return nil on syntax error")
assert(not string.find(err2, "@", 1, true), "loadfile error should not contain @, got: " .. tostring(err2))

os.remove(tmpfile)
print("OK")
