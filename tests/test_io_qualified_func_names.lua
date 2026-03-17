-- io.write / io.read errors should use qualified names like 'io.write'.
-- Currently golua uses short names 'write' / 'read'.

-- io.write with nil should say 'io.write'
local ok1, err1 = pcall(io.write, nil)
assert(not ok1)
assert(err1:find("'io%.write'"), "io.write error should say 'io.write', got: " .. err1)

-- io.read with boolean should say 'io.read'
local ok2, err2 = pcall(io.read, true)
assert(not ok2)
assert(err2:find("'io%.read'"), "io.read error should say 'io.read', got: " .. err2)
