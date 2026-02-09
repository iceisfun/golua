-- table.unpack tests
do
    print(pcall(table.unpack))
    --> ~^false\t.*table expected

    print(pcall(table.unpack, 1))
    --> ~^false\t.*table expected

    print(pcall(table.unpack, {}, "a"))
    --> ~^false\t.*number expected

    print(pcall(table.unpack, {}, 2, "a"))
    --> ~^false\t.*number expected

    print(table.unpack({3, 4, 1, 5}))
    --> =3	4	1	5

    print(table.unpack({1, 2, 3, 4, 5, 6}, 3, 5))
    --> =3	4	5

    print(table.unpack({3, 2, 1}, 3, 5))
    --> =1	nil	nil

    print(table.unpack({4, 3, 2}, -1, 1))
    --> =nil	nil	4

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g") end
    })
    print(pcall(table.unpack, tt))
    --> ~false\t.* g
end
