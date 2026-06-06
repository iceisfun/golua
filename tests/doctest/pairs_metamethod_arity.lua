-- Lua 5.5: pairs returns four values -- the fourth is the to-be-closed
-- value for the generic-for loop. Values from __pairs beyond the fourth
-- are dropped.

do
    local t = setmetatable({}, {
        __pairs = function()
            -- no return values
        end
    })
    print(pcall(pairs, t))
    --> =true	nil	nil	nil	nil
end

do
    local t = setmetatable({}, {
        __pairs = function()
            return 1
        end
    })
    print(pairs(t))
    --> =1	nil	nil	nil
end

do
    local t = setmetatable({}, {
        __pairs = function()
            return 1, 2, 3, 4
        end
    })
    print(pairs(t))
    --> =1	2	3	4
end
