-- Bug: table.concat error message always says "nil" for the type
-- instead of the actual type of the invalid value.

-- Test 1: boolean in table should report "boolean"
local ok1, err1 = pcall(table.concat, {1, true, 3}, ",")
assert(not ok1, "should error on boolean")
assert(err1:find("boolean"), "error should mention 'boolean', got: " .. err1)

-- Test 2: table in table should report "table"
local ok2, err2 = pcall(table.concat, {"a", {}, "c"}, ",")
assert(not ok2, "should error on table")
assert(err2:find("table"), "error should mention 'table', got: " .. err2)

-- Test 3: function in table should report "function"
local ok3, err3 = pcall(table.concat, {"a", print, "c"}, ",")
assert(not ok3, "should error on function")
assert(err3:find("function"), "error should mention 'function', got: " .. err3)

print("PASS")
