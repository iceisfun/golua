-- Direct calls to package.searchers still need the same string argument checks
-- that package.searchpath/package.searcher paths apply in Lua 5.4.

for i = 1, 4 do
    local ok, err = pcall(function()
        return package.searchers[i]({})
    end)
    print(i, ok, err:find("string expected, got table", 1, true) ~= nil)
    --> =1	false	true
    --> =2	false	true
    --> =3	false	true
    --> =4	false	true
end
