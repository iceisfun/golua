local sink = {}
local target = coroutine.create(function()
end)

debug.setmetatable(target, {__newindex = sink})

local t = setmetatable({}, {__newindex = target})
t.x = 7

print(sink.x)
--> =7
