-- test_load_env_binding_contract: load() _ENV binding should follow Lua 5.4

-- Omitted env uses globals
x = 42
local f0 = assert(load("return x", "chunk0", "t"))
assert(f0() == 42, "load with omitted env should use globals")

-- Explicit table env uses that table
local env = {x = 7}
local f1 = assert(load("return x", "chunk1", "t", env))
assert(f1() == 7, "load should use provided table env")

-- Explicit nil env binds _ENV=nil (not globals)
local f2 = assert(load("return x", "chunk2", "t", nil))
local ok2, err2 = pcall(f2)
assert(ok2 == false, "load with explicit nil env should fail when accessing globals")
assert(type(err2) == "string" and err2:find("attempt to index"),
       "unexpected error for nil env: " .. tostring(err2))

-- Explicit non-table env binds _ENV to that value
local f3 = assert(load("return x", "chunk3", "t", 1))
local ok3, err3 = pcall(f3)
assert(ok3 == false, "load with numeric env should fail when indexing _ENV")
assert(type(err3) == "string" and err3:find("attempt to index"),
       "unexpected error for numeric env: " .. tostring(err3))
