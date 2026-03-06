-- string.format spec length limit should match Lua 5.4
-- Bug: GoLua allowed 50-char specs, Lua 5.4 limits to ~22

-- 20 zeros between % and d should work (first 0 is flag, rest are width 0)
assert(string.format("%" .. string.rep("0", 20) .. "d", 1) == "1")

-- 21 zeros should fail with "too long"
local ok, err = pcall(string.format, "%" .. string.rep("0", 21) .. "d", 1)
assert(not ok, "21 zeros should be too long")
assert(string.find(err, "too long"), "wrong error: " .. tostring(err))

-- Same for other conversions
ok, err = pcall(string.format, "%" .. string.rep("0", 21) .. "f", 1.0)
assert(not ok, "21 zeros with %f should be too long")

print("PASS")
