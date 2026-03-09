-- Test: loadfile and dofile can load binary (precompiled) chunks

-- Create a simple function and dump it
local chunk = string.dump(function() return 42 end)

-- Write binary chunk to a temp file
local tmpfile = os.tmpname()
local f = io.open(tmpfile, "wb")
assert(f, "failed to create temp file")
f:write(chunk)
f:close()

-- Test loadfile with binary chunk
local fn, err = loadfile(tmpfile)
assert(fn, "loadfile failed: " .. tostring(err))
local result = fn()
assert(result == 42, "expected 42, got " .. tostring(result))

-- Test loadfile with binary mode
local fn2, err2 = loadfile(tmpfile, "b")
assert(fn2, "loadfile mode='b' failed: " .. tostring(err2))
assert(fn2() == 42)

-- Test loadfile with text-only mode rejects binary
local fn3, err3 = loadfile(tmpfile, "t")
assert(fn3 == nil, "expected loadfile to reject binary with mode='t'")
assert(err3:find("binary chunk"), "expected binary chunk error: " .. tostring(err3))

-- Test dofile with binary chunk
local result2 = dofile(tmpfile)
assert(result2 == 42, "dofile expected 42, got " .. tostring(result2))

-- Clean up
os.remove(tmpfile)

print("OK")
