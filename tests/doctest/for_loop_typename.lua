-- for-loop type errors should use __name for type name
-- Lua 5.4 uses the __name metafield from metatables for error messages

-- Tables with __name should show __name in for-loop errors
do
    local mt = {__name = "MyType"}
    local obj = setmetatable({}, mt)
    local ok, err = pcall(function()
        for i = obj, 10 do end
    end)
    -- Error should say "got MyType" not "got table"
    print(err:match("got (.+)%)"))
    --> =MyType
end
