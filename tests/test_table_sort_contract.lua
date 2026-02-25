-- test_table_sort_contract: table.sort should enforce Lua 5.4 argument/comparison rules

-- Mixed incomparable values must error in default sort
local ok1, err1 = pcall(function()
    local t = {1, "a"}
    table.sort(t)
end)
assert(ok1 == false, "table.sort with mixed types should fail")
assert(type(err1) == "string" and err1:find("attempt to compare"),
       "unexpected mixed-type sort error: " .. tostring(err1))

-- Comparator must be a function when provided
local ok2, err2 = pcall(function()
    local t = {3, 1, 2}
    table.sort(t, 1)
end)
assert(ok2 == false, "table.sort comparator must be a function")
assert(type(err2) == "string" and err2:find("bad argument #2") and err2:find("function expected"),
       "unexpected comparator-arg error: " .. tostring(err2))

-- Control: valid custom comparator still works
local t = {3, 1, 2}
table.sort(t, function(a, b) return a > b end)
assert(t[1] == 3 and t[2] == 2 and t[3] == 1, "custom comparator sort should still work")
