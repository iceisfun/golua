-- Test: table.sort with alternating comparator matches Lua 5.4 behavior
-- An alternating comparator is technically invalid (not strict weak ordering),
-- but Lua 5.4 does not detect it for certain array sizes. GoLua should match.

-- Helper: test that sort with alternating comparator does NOT error
local function check_no_error(n)
    local t = {}
    for i = 1, n do t[i] = i end
    local c = 0
    local ok, err = pcall(table.sort, t, function(a, b)
        c = c + 1
        return c % 2 == 0
    end)
    if not ok then
        error("n=" .. n .. ": unexpected error: " .. tostring(err))
    end
end

-- These sizes should NOT error (verified against Lua 5.4)
check_no_error(2)
check_no_error(3)
check_no_error(4)
check_no_error(8)

-- Lua 5.4 DOES error for n=5 with this comparator, so we test that too
do
    local t = {3,1,4,1,5}
    local c = 0
    local ok, err = pcall(table.sort, t, function(a,b) c=c+1; return c%2==0 end)
    assert(not ok, "n=5: expected error but got none")
    assert(tostring(err):find("invalid order function"),
        "n=5: expected 'invalid order function' error, got: " .. tostring(err))
end

-- Also test the original bug report case
do
    local t = {3,1,4,1,5,9,2,6}
    local c = 0
    local ok, err = pcall(table.sort, t, function(a,b) c=c+1; return c%2==0 end)
    if not ok then
        error("n=8 (bug report case): unexpected error: " .. tostring(err))
    end
end

print("OK")
