-- Test: load() with empty string chunk name should produce [string ""] not [string "[string ""]"]
-- Bug: GoLua double-wraps the empty chunk name

local f = load("error('boom')", "")
local ok, err = pcall(f)

-- Lua 5.4 produces: [string ""]:1: boom
-- GoLua produces: [string "[string ""]"]:1: boom
assert(err == '[string ""]:1: boom', "got: " .. tostring(err))
print("PASS")
