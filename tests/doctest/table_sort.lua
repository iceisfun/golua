-- table.sort tests
do
    print(pcall(table.sort))
    --> ~^false\t.*table expected

    print(pcall(table.sort, 1))
    --> ~^false\t.*table expected

    local t = {3, 2, 4, 1, 5}
    table.sort(t)
    print(table.concat(t))
    --> =12345

    table.sort(t, function(x, y) return x > y end)
    print(table.concat(t))
    --> =54321

    local t = {"bar", "bat", "fur", "ball", "four"}
    table.sort(t)
    print(table.concat(t, " "))
    --> =ball bar bat four fur

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g", 0) end
    })
    print(pcall(table.sort, tt))
    --> =false	g

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function(n, i) return -i end,
        __newindex=function() error("s", 0) end
    })
    print(pcall(table.sort, tt))
    --> =false	s
end
