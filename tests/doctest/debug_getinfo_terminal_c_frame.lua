-- Lua 5.4 exposes a synthetic [C] frame at the bottom of the main call stack.
-- debug.getinfo at the level past all real frames should return a C frame.

local levels = {}
local i = 0
while true do
    local info = debug.getinfo(i, "Sl")
    if not info then break end
    levels[#levels + 1] = info.what
    i = i + 1
end

-- The last frame should be a C frame (the runtime entry point)
print(levels[#levels])
--> =C
-- There should be at least 2 levels (main chunk + terminal C)
print(#levels >= 2)
--> =true
