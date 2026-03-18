-- test_table_unpack_limits: pathological unpack ranges should fail fast

local ok, err = pcall(function()
    return table.unpack({1, 2}, 1, 2147483648)
end)
assert(ok == false, "table.unpack huge range should error")
assert(type(err) == "string" and err:find("too many results to unpack"),
       "expected 'too many results to unpack', got: " .. tostring(err))

-- 1,000,001 results should also error (matching Lua's LUAI_MAXSTACK limit)
local ok2, err2 = pcall(function()
    return table.unpack({}, 1, 1000001)
end)
assert(ok2 == false, "table.unpack 1M+ range should error")
assert(type(err2) == "string" and err2:find("too many results to unpack"),
       "expected 'too many results to unpack', got: " .. tostring(err2))

-- Adjacent: reversed range should return no values
local t = {table.unpack({1, 2, 3}, 3, 2)}
assert(#t == 0, "reversed unpack range should produce zero results")
