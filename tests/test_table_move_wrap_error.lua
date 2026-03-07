-- Bug: table.move wrap-around error is not formatted correctly.
-- GoLua says: "destination wrap around"
-- Lua 5.4 says: "bad argument #4 to 'table.move' (destination wrap around)"

local ok, err = pcall(table.move, {}, 1, 5, math.maxinteger - 2)
assert(ok == false, "table.move should fail on wrap-around")
assert(err:find("bad argument"), "error should start with 'bad argument', got: " .. tostring(err))
assert(err:find("destination wrap around"), "error should mention 'destination wrap around', got: " .. tostring(err))

