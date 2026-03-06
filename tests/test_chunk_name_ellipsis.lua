-- Chunk names for multi-line load() should include "..." suffix
-- Bug: GoLua omitted "..." when source was truncated at first newline

local f = load("local x\nreturn nil + 1")
local ok, err = pcall(f)
assert(not ok)
assert(string.find(err, '%[string "local x%.%.%."%]'),
    'expected [string "local x..."] in error, got: ' .. tostring(err))

-- Single-line source should NOT have "..."
local f2 = load("return nil + 1")
local ok2, err2 = pcall(f2)
assert(not ok2)
assert(string.find(err2, '%[string "return nil %+ 1"%]'),
    'expected [string "return nil + 1"] in error, got: ' .. tostring(err2))

print("PASS")
