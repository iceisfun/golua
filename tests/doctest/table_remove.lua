-- table.remove tests
do
    print(pcall(table.remove))
    --> ~^false\t.*table expected

    print(pcall(table.remove, 1))
    --> ~^false\t.*table expected

    print(pcall(table.remove, {}, true))
    --> ~^false\t.*number expected

    -- Lua 5.4: table.remove({}, 2) errors (pos=2 != size=0, validate fails)
    print(pcall(table.remove, {}, 2))
    --> ~^false\t.*out of bounds

    local t = {1, 2, 3, 4, 5}

    print(#t)
    --> =5

    print(table.remove(t))
    --> =5

    print(#t)
    --> =4

    print(table.remove(t))
    --> =4

    print(t[4])
    --> =nil

    print(#t)
    --> =3

    print(table.remove(t, 2))
    --> =2

    print(table.concat(t))
    --> =13

    -- Lua 5.4: table.remove({}) defaults pos=size=0, pos==size so no validation
    print(table.remove({}))
    --> =nil

    -- Lua 5.4: table.remove({}, 0) — pos=0, size=0, pos==size → no validation → nil
    print(table.remove({}, 0))
    --> =nil

    -- Lua 5.4: table.remove({}, 1) — pos=1 = size+1=1, valid → nil
    print(pcall(table.remove, {}, 1))
    --> =true	nil

    -- Lua 5.4: table.remove(t, #t+1) — pos=3 = size+1=3, valid → nil
    print(pcall(table.remove, t, 3))
    --> =true	nil

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function() error("g", 0) end
    })
    print(pcall(table.remove, tt))
    --> =false	g

    local tt = {}
    setmetatable(tt, {
        __len=function() return 3 end,
        __index=function(n, i) return -i end,
        __newindex=function() error("s", 0) end
    })
    print(pcall(table.remove, tt))
    --> =false	s

    print(pcall(table.remove, tt, 2))
    --> =false	s
end
