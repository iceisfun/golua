-- Test: table.sort detects invalid order function

local count = 0
local t = {1,2,3,4,5}
local ok, err = pcall(table.sort, t, function(a,b)
    count = count + 1
    return count % 2 == 0
end)
assert(ok == false, "expected error from invalid order function, got ok=true")
assert(type(err) == "string" and err:find("invalid order function"),
    "expected 'invalid order function' error, got: " .. tostring(err))

print("OK")
