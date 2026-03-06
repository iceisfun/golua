-- __close with non-callable value should include (metamethod 'close') in error
local t1 = setmetatable({}, {__close = "not a function"})
local ok1, err1 = pcall(function()
    local x <close> = t1
end)
assert(not ok1)
assert(err1:find("attempt to call a string value"), err1)
assert(err1:find("(metamethod 'close')", 1, true), "missing metamethod context: " .. err1)

local t2 = setmetatable({}, {__close = 42})
local ok2, err2 = pcall(function()
    local x <close> = t2
end)
assert(not ok2)
assert(err2:find("attempt to call a number value"), err2)
assert(err2:find("(metamethod 'close')", 1, true), "missing metamethod context: " .. err2)

local t3 = setmetatable({}, {__close = true})
local ok3, err3 = pcall(function()
    local x <close> = t3
end)
assert(not ok3)
assert(err3:find("attempt to call a boolean value"), err3)
assert(err3:find("(metamethod 'close')", 1, true), "missing metamethod context: " .. err3)

print("OK")
