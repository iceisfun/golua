-- or/and expressions should suppress variable name annotations in errors
-- When a value comes through or/and, the name may not be accurate

do
    local ok, err = pcall(load("aaa={}; (aaa or aaa)()"))
    -- Should NOT show "(global 'aaa')" since it came through 'or'
    print(err)
    --> ~attempt to call a table value$
end

do
    local ok, err = pcall(load("aaa={}; (aaa and aaa)()"))
    -- Should NOT show "(global 'aaa')" since it came through 'and'
    print(err)
    --> ~attempt to call a table value$
end
