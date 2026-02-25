-- test_table_unpack_limits: pathological unpack ranges should fail fast

local ok, err = pcall(function()
    return table.unpack({1, 2}, 1, 2147483648)
end)
assert(ok == false, "table.unpack huge range should error")
assert(type(err) == "string" and err:find("too many results to unpack"),
       "expected 'too many results to unpack', got: " .. tostring(err))

-- Adjacent: reversed range should return no values
local t = {table.unpack({1, 2, 3}, 3, 2)}
assert(#t == 0, "reversed unpack range should produce zero results")
