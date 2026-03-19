-- debug.getinfo(..., "n") should preserve the active __close call name on a
-- normal scope exit.

local function inspect(level)
    local i = debug.getinfo(level, "n")
    print(tostring(i and i.name), i and i.namewhat or "nil")
end

local function probe()
    local obj = setmetatable({}, {
        __close = function()
            inspect(2)
        end,
    })
    local x <close> = obj
end

probe()
--> =close	metamethod
