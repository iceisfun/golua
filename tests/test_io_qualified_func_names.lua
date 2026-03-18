-- io.write / io.read error names should be resolved from the call site
-- bytecode, matching Lua 5.4's luaL_argerror behavior. When called as
-- io.write(), the name 'write' is resolved from GETFIELD on the io table.

-- io.write with nil argument → name resolved to 'write'
local ok1, err1 = pcall(io.write, nil)
assert(not ok1)
assert(err1:find("'write'"), "io.write error should say 'write', got: " .. err1)

-- io.read with boolean argument → name resolved to 'read'
local ok2, err2 = pcall(io.read, true)
assert(not ok2)
assert(err2:find("'read'"), "io.read error should say 'read', got: " .. err2)
