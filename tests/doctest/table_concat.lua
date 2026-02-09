-- table.concat tests
do
    print(pcall(table.concat))
    --> ~^false\t.*table expected

    print(pcall(table.concat, "not a table"))
    --> ~^false\t.*table expected

    -- Lua 5.4 coerces number separator to string (result is empty string)
    print(table.concat({}, 1))
    --> =

    print(pcall(table.concat, {}, "--", false))
    --> ~^false\t.*number expected

    print(pcall(table.concat, {}, "--", 1, {}))
    --> ~^false\t.*number expected

    local t = {1, 2, 3}

    print(table.concat(t))
    --> =123

    print(table.concat(t, "--"))
    --> =1--2--3

    print(table.concat({}))
    --> =

    print(type(table.concat({})))
    --> =string

    print(table.concat({"foo"}))
    --> =foo

    print(table.concat(t, "", 2, 3))
    --> =23

    print(table.concat(t, "", 2, 2))
    --> =2

    print(table.concat(t, "", 3, 2))
    --> =

    t[-1]="hel"
    t[0]="lo"
    print(table.concat(t, "", -1, 1))
    --> =hello1

    print(pcall(table.concat, t, "", 2, 5))
    --> ~^false\t.*

    print(pcall(table.concat, {}, " ", 10, 10))
    --> ~false\t.*at index 10
end
