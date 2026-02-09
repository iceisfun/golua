-- string.gmatch should support optional init argument (Lua 5.4)
local t = {}
for w in string.gmatch("hello world from Lua", "%a+", 8) do
    t[#t + 1] = w
end
assert(t[1] == "orld", "gmatch init should start at position 8, got: " .. tostring(t[1]))
assert(#t == 3, "expected 3 words after position 8")

-- Negative init
local t2 = {}
for w in string.gmatch("hello world from Lua", "%a+", -7) do
    t2[#t2 + 1] = w
end
assert(t2[1] == "rom", "gmatch negative init")
