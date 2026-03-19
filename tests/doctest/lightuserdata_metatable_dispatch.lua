-- Once a lightuserdata metatable is installed, normal metamethod dispatch
-- should work for indexing and pairs() just like Lua 5.4.

local function outer()
    local x = 1
    return function() return x end
end

local id = debug.upvalueid(outer(), 1)
debug.setmetatable(id, {
    __index = { foo = 42 },
    __pairs = function()
        return next, { a = 1 }, nil
    end,
})

local ok1, res1 = pcall(function() return id.foo end)
print(ok1, res1)
--> =true	42

local ok2, res2 = pcall(function()
    local out = ""
    for k, v in pairs(id) do
        out = out .. k .. v
    end
    return out
end)
print(ok2, res2)
--> =true	a1
