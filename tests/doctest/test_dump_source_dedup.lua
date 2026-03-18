-- string.dump should not duplicate source names in sub-functions
local name = string.rep("x", 1000)
local prog = "local function f() return 1 end; return f"
local p = assert(load(prog, name))
local size = #string.dump(p)
-- Lua 5.4 produces ~1156 bytes; without dedup it's ~2109+
-- Allow some tolerance but ensure it's under 1500
print(size < 1500)
--> true
