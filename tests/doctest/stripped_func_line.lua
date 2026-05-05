-- Stripped function errors should show line "?"
-- When a function has no debug info, Lua 5.5 prints "?" as the line number
-- (Lua 5.4 used -1; this changed in 5.5).

do
    local f = function(a) return a + 1 end
    f = assert(load(string.dump(f, true)))
    local ok, err = pcall(f, {})
    print(err)
    --> =?:?: attempt to perform arithmetic on a table value
end
