-- pairs(__pairs) must always return exactly three values.

do
    local t = setmetatable({}, {
        __pairs = function()
            -- no return values
        end
    })
    print(pcall(pairs, t))
    --> =true	nil	nil	nil
end

do
    local t = setmetatable({}, {
        __pairs = function()
            return 1
        end
    })
    print(pairs(t))
    --> =1	nil	nil
end

do
    local t = setmetatable({}, {
        __pairs = function()
            return 1, 2, 3, 4
        end
    })
    print(pairs(t))
    --> =1	2	3
end
