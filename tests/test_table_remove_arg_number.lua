-- Test: table.remove error should reference argument #2 (position), not #1
local ok, err = pcall(table.remove, {1,2,3}, 10)
assert(not ok)
assert(err:find("#2"), "expected error to reference arg #2, got: " .. tostring(err))
print("OK")
