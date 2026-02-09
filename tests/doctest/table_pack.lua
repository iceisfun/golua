-- table.pack tests
do
    local t = table.pack(3, 2, 1, 4, 5)
    print(t.n, #t)
    --> =5	5
    print(table.concat(t))
    --> =32145
end
