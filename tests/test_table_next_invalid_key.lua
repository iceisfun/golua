-- Tests that next() throws an error when given an invalid key that is not in the table.
-- Lua 5.4 reference manual: "If `index` is present, it must be a valid key of the table."
-- It should raise an error "invalid key to 'next'".

local t = {A = 1, B = 2}

-- Test with a key that was never in the table
local ok, err = pcall(next, t, "Z")
if ok then
    error("next with invalid key 'Z' should have raised an error, but it succeeded")
end

if type(err) == "string" and not string.find(err, "invalid key to 'next'") then
    error("expected 'invalid key to 'next'' error, got: " .. tostring(err))
end

-- Test with an invalid integer key
local ok2, err2 = pcall(next, t, 999)
if ok2 then
    error("next with invalid integer key 999 should have raised an error")
end
