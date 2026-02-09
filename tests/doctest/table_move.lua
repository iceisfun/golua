-- table.move tests
do
    print(pcall(table.move))
    --> ~^false\t.*table expected

    print(pcall(table.move, 1, 2, 3, 4))
    --> ~^false\t.*table expected

    print(pcall(table.move, {}, true, 3, 4))
    --> ~^false\t.*number expected

    print(pcall(table.move, {}, 2, false, 4))
    --> ~^false\t.*number expected

    print(pcall(table.move, {}, 2, 3, "xxx"))
    --> ~^false\t.*number expected

    print(pcall(table.move, {}, 1, 2, 3, "bar"))
    --> ~^false\t.*table expected

    local t = {1, 2, 3, 4}
    table.move(t, 2, 4, 3)
    print(table.concat(t))
    --> =12234

    table.move(t, 3, 5, 2)
    print(table.concat(t))
    --> =12344

    local u = {}
    print(table.concat(table.move(t, 1, 4, 1, u)))
    --> =1234

    -- Edge conditions
    print(pcall(table.move, {}, 0, 10, math.maxinteger - 9))
    --> ~false\t.*wrap around

    print(pcall(table.move, {}, -10, math.maxinteger, -20))
    --> ~false\t.*interval too large
end
