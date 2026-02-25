-- Bugs #8 and #9: error source formatting issues
-- #9: pcall(error, "test") adds position info when called through C frames with no Lua source
-- #8: source name should be wrapped in [string "..."] format (tested via load chunkname)

-- Bug #9: error called from C frame, level 1 has no Lua source — should not add position
local ok, e = pcall(error, "test")
assert(e == "test", "pcall(error, 'test') should be plain 'test', got: " .. tostring(e))

-- Bug #8/#12: load chunkname should be wrapped in [string "..."] format
local f = load("error('x')", "test_chunk")
local ok2, e2 = pcall(f)
assert(e2:find('%[string "test_chunk"%]'), "chunkname not wrapped: " .. tostring(e2))

-- = prefix in chunkname means use literally (strip the = for display)
local f2 = load("error('x')", "=mychunk")
local ok3, e3 = pcall(f2)
assert(e3:find('mychunk:'), "= prefix chunkname should be used literally: " .. tostring(e3))
