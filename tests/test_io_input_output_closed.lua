-- io.input() and io.output() should reject closed file handles.
-- Lua 5.4 raises "attempt to use a closed file".

local f = io.tmpfile()
f:close()

local ok1, err1 = pcall(io.input, f)
assert(not ok1, "io.input should reject closed file")
assert(tostring(err1):find("closed"), "should mention closed, got: " .. tostring(err1))

local ok2, err2 = pcall(io.output, f)
assert(not ok2, "io.output should reject closed file")
assert(tostring(err2):find("closed"), "should mention closed, got: " .. tostring(err2))
