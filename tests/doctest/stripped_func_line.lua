-- Stripped function errors should show line -1
-- When a function has no debug info, errors should use -1 as line number

do
    local f = function(a) return a + 1 end
    f = assert(load(string.dump(f, true)))
    local ok, err = pcall(f, {})
    print(err)
    --> =?:-1: attempt to perform arithmetic on a table value
end
