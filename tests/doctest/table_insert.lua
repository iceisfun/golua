-- table.insert tests
do
    print(pcall(table.insert))
    --> ~^false\t.*table expected

    print(pcall(table.insert, 1, false))
    --> ~^false\t.*table expected

    print(pcall(table.insert, {}, "hello", true))
    --> ~^false\t.*number expected

    local t = {1, 2, 3}
    table.insert(t, "foo")
    print(t[4])
    --> =foo

    table.insert(t, 2, 42)
    print(t[2], t[3], #t)
    --> =42	2	5

    print(pcall(table.insert, t, -1, 1))
    --> ~^false\t.*

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g", 0) end
    })
    print(pcall(table.insert, tt, 1, 12))
    --> =false	g

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() return 2 end,
        __newindex=function() error("s", 0) end
    })
    print(pcall(table.insert, tt, 1, 12))
    --> =false	s

    print(pcall(table.insert, tt, 4, 123))
    --> =false	s
end
