-- Bugs #11 and #12: load() edge cases

-- Bug #11: load(nil) should error, not return nil quietly
local ok, e = pcall(load, nil)
assert(not ok, "load(nil) should error, not return nil")

-- Bug #12: load chunkname should be wrapped in [string "..."] format
local f = load("error('x')", "mychunk")
local ok2, e2 = pcall(f)
assert(e2:find('%[string "mychunk"%]'), 'expected [string "mychunk"] in: ' .. tostring(e2))

-- = prefix in chunkname means use literally
local f2 = load("error('x')", "=mychunk")
local ok3, e3 = pcall(f2)
assert(e3:find('mychunk:'), "= prefix chunkname: " .. tostring(e3))
