-- Test: Lua 5.4 reports table.remove position range errors on argument #1.
local ok, err = pcall(table.remove, {1,2,3}, 10)
assert(not ok)
assert(err:find("#1"), "expected error to reference arg #1, got: " .. tostring(err))
print("OK")
