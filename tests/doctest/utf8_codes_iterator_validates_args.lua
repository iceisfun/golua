-- Calling the utf8.codes iterator closure directly should still validate the
-- subject string and state arguments like Lua 5.4 does.

local it = utf8.codes("A")
local ok, err = pcall(function()
    return it({}, {})
end)

print(ok, err:find("string expected, got table", 1, true) ~= nil)
--> =false	true
