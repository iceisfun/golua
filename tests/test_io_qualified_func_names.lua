-- io.write / io.read error names should match Lua 5.4 behavior.
-- When called through pcall (no Lua caller frame), Lua 5.4 resolves
-- the function name via pushglobalfuncname, yielding 'io.write'/'io.read'.
-- When called directly, bytecode resolution gives 'write'/'read'.

-- io.write with nil argument via pcall → name resolved to 'io.write'
local ok1, err1 = pcall(io.write, nil)
assert(not ok1)
assert(err1:find("'io.write'"), "io.write error should say 'io.write', got: " .. err1)

-- io.read with boolean argument via pcall → name resolved to 'io.read'
local ok2, err2 = pcall(io.read, true)
assert(not ok2)
assert(err2:find("'io.read'"), "io.read error should say 'io.read', got: " .. err2)
