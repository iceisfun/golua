-- Test: ipairs iterator on non-table gives "attempt to index" error

local ok, err = pcall(function() for k,v in ipairs(42) do end end)
assert(ok == false, "expected error")
assert(type(err) == "string" and err:find("attempt to index a number value"),
    "expected 'attempt to index a number value', got: " .. tostring(err))

-- Normal usage still works
local t = {10, 20, 30}
local sum = 0
for i, v in ipairs(t) do sum = sum + v end
assert(sum == 60, "ipairs should work normally on tables")

print("OK")
