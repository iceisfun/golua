-- Test that file read/write errors return nil, msg, errno (3 values)
-- Lua 5.4: read/write on invalid file descriptors returns nil, string, number

local fname = os.tmpname()
local f = io.open(fname, "w")
f:write("test content\n")
f:close()

local function ismsg(m)
  return type(m) == "string" and not tonumber(m)
end

-- Read from write-only file
f = io.open(fname, "w")
local r, m, c = f:read()
assert(not r, "read should fail")
assert(ismsg(m), "error message should be a string: " .. tostring(m))
assert(type(c) == "number", "errno should be a number, got " .. type(c))
f:close()
print("read from write-only: OK")

-- Write to read-only file
f = io.open(fname, "r")
r, m, c = f:write("whatever")
assert(not r, "write should fail")
assert(ismsg(m), "error message should be a string: " .. tostring(m))
assert(type(c) == "number", "errno should be a number, got " .. type(c))
f:close()
print("write to read-only: OK")

-- Lines iterator from write-only file
f = io.open(fname, "w")
r, m = pcall(f:lines())
assert(r == false, "lines should fail")
assert(ismsg(m), "error message should be a string: " .. tostring(m))
f:close()
print("lines from write-only: OK")

-- Default input/output closed errors
io.output(fname)
io.write("test")
io.close()
local ok, err = pcall(io.write, "x")
assert(not ok)
assert(string.find(err, "output file is closed", 1, true),
  "expected 'output file is closed', got: " .. tostring(err))
print("closed output error: OK")

os.remove(fname)
print("PASS")
